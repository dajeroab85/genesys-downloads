package functions

import (
	"fmt"
	"sync"
	"time"

	"github.com/goDownloadRecording/logger"
	sdk "github.com/mypurecloud/platform-client-sdk-go/v157/platformclientv2"
	"go.uber.org/zap"
)

const (
	maxRetries   = 3
	retryDelay   = 2 * time.Second
	maxBatchSize = 100
)

// AddConversationRecordingsToBatch consulta metadata de grabaciones en paralelo y construye el batch.
func AddConversationRecordingsToBatch(conversationIDs []string, recordingApi *sdk.RecordingApi, workers int) ([]sdk.Batchdownloadrequest, error) {
	var (
		batchRequests  []sdk.Batchdownloadrequest
		mu             sync.Mutex
		wg             sync.WaitGroup
		conversationCh = make(chan string, len(conversationIDs))
	)

	// Cargar todas las conversationIDs al canal
	for _, id := range conversationIDs {
		conversationCh <- id
	}
	close(conversationCh)

	// Pool de workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for conversationID := range conversationCh {
				logger.Log.Debug("🧵 Worker fetching metadata",
					zap.Int("WorkerID", workerID),
					zap.String("ConversationID", conversationID))

				var recordingsData []sdk.Recordingmetadata
				var err error

				for attempt := 1; attempt <= maxRetries; attempt++ {
					var response []sdk.Recordingmetadata
					response, _, err = recordingApi.GetConversationRecordingmetadata(conversationID)
					if err == nil {
						recordingsData = response
						break
					}

					logger.Log.Warn("Retrying metadata fetch...",
						zap.String("ConversationID", conversationID),
						zap.Int("Attempt", attempt),
						zap.Error(err))

					time.Sleep(retryDelay * time.Duration(attempt))
				}

				if err != nil {
					logger.Log.Error("Failed to fetch metadata after retries",
						zap.String("ConversationID", conversationID),
						zap.Error(err))
					continue
				}

				var localBatch []sdk.Batchdownloadrequest
				for _, recording := range recordingsData {
					if recording.Id != nil && recording.ConversationId != nil {
						localBatch = append(localBatch, sdk.Batchdownloadrequest{
							ConversationId: recording.ConversationId,
							RecordingId:    recording.Id,
						})
						logger.Log.Debug("Added recording",
							zap.String("ConversationID", *recording.ConversationId),
							zap.String("RecordingID", *recording.Id))
					}
				}

				// Agregar al batch global con mutex
				mu.Lock()
				batchRequests = append(batchRequests, localBatch...)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if len(batchRequests) == 0 {
		logger.Log.Warn("No recordings found to include in the batch.")
		return nil, nil
	}

	logger.Log.Info("✅ Batch ready", zap.Int("TotalRecordings", len(batchRequests)))
	return batchRequests, nil
}

// SendBatchRequests divide las solicitudes en lotes y los envía por separado
func SendBatchRequests(recordApi *sdk.RecordingApi, batchRequests []sdk.Batchdownloadrequest) ([]*sdk.Batchdownloadjobsubmissionresult, error) {
	if len(batchRequests) == 0 {
		logger.Log.Warn("Empty batch request. No recordings to submit.")
		return nil, nil
	}

	var results []*sdk.Batchdownloadjobsubmissionresult

	for i := 0; i < len(batchRequests); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(batchRequests) {
			end = len(batchRequests)
		}

		chunk := batchRequests[i:end]
		chunkCopy := chunk // evitar referencia al slice original

		batchRequest := sdk.Batchdownloadjobsubmission{
			BatchDownloadRequestList: &chunkCopy,
		}

		resp, _, err := recordApi.PostRecordingBatchrequests(batchRequest)
		if err != nil {
			logger.Log.Error("❌ Error sending partial batch request",
				zap.Int("Start", i), zap.Int("End", end), zap.Error(err))
			continue
		}

		logger.Log.Info("✅ Partial batch sent",
			zap.String("BatchID", *resp.Id),
			zap.Int("Count", len(chunkCopy)))

		results = append(results, resp)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no batch requests were successfully sent")
	}

	return results, nil
}

// PollBatchJobUntilReady consulta hasta que el batch esté listo
func PollBatchJobUntilReady(recordApi *sdk.RecordingApi, jobID string, maxRetries int, delay time.Duration) (*sdk.Batchdownloadjobstatusresult, error) {
	var (
		result       *sdk.Batchdownloadjobstatusresult
		err          error
		lastProgress = -1
		stalledCount = 0
		maxStalled   = 30
	)

	for i := 0; i < maxRetries; i++ {
		result, _, err = recordApi.GetRecordingBatchrequest(jobID)
		if err != nil {
			logger.Log.Error("Error polling batch request status", zap.String("BatchID", jobID), zap.Error(err))
			return nil, err
		}

		curr := getInt(result.ResultCount)
		expected := getInt(result.ExpectedResultCount)

		if curr == expected && expected > 0 {
			logger.Log.Info("✅ Batch job completed", zap.String("BatchID", jobID), zap.Int("TotalRecordings", expected))

			if result.Results != nil {
				for _, item := range *result.Results {
					if item.ResultUrl != nil {
						fmt.Println("🔗 Download URL:", *item.ResultUrl)
					} else if item.ErrorMsg != nil && *item.ErrorMsg != "" {
						logger.Log.Warn("⚠️ Grabación fallida", zap.String("RecordingId", getString(item.RecordingId)), zap.String("Error", *item.ErrorMsg))
					}

				}
			}
			return result, nil
		}

		if curr == lastProgress {
			stalledCount++
			logger.Log.Debug("No progress in batch job", zap.String("BatchID", jobID), zap.Int("StalledCount", stalledCount))
		} else {
			stalledCount = 0
		}
		lastProgress = curr

		if stalledCount >= maxStalled {
			logger.Log.Warn("⚠️ Batch job estancado. Cancelando espera.", zap.String("BatchID", jobID))
			break
		}

		logger.Log.Info("⏳ Waiting for batch job...", zap.String("BatchID", jobID), zap.Int("Progress", curr), zap.Int("Expected", expected))
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("batch job %s did not complete in time", jobID)
}

func PollAllBatchesInParallel(recordApi *sdk.RecordingApi, results []*sdk.Batchdownloadjobsubmissionresult) {
	var wg sync.WaitGroup
	for _, res := range results {
		if res.Id == nil {
			continue
		}
		wg.Add(1)
		go func(jobID string) {
			defer wg.Done()
			_, err := PollBatchJobUntilReady(recordApi, jobID, 40, 15*time.Second)
			if err != nil {
				logger.Log.Error("Error polling batch job", zap.String("BatchID", jobID), zap.Error(err))
			}
		}(*res.Id)
	}
	wg.Wait()
}

func getString(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getInt(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}
