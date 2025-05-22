package main

import (
	//"fmt"
	"sync"
	"time"

	"github.com/goDownloadRecording/config"
	query "github.com/goDownloadRecording/conversation_query"
	"github.com/goDownloadRecording/functions"
	"github.com/goDownloadRecording/logger"
	sdk "github.com/mypurecloud/platform-client-sdk-go/v157/platformclientv2"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()

	logger.Log.Info("Starting Genesys Download Recording")

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Log.Error("Error loading environment variables", zap.Error(err))
		return
	}

	// Configuración y autorización del SDK
	config := sdk.GetDefaultConfiguration()
	config.BasePath = "https://api." + cfg.GenesysCloudEnvironment
	downloadPath := "./recordings/"
	err = config.AuthorizeClientCredentials(cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		logger.Log.Error("Error authorizing client credentials", zap.Error(err))
		return
	}

	// Instanciar APIs
	analyticsApi := sdk.NewAnalyticsApi()
	recordApi := sdk.NewRecordingApi()

	// Construir la query
	logger.Log.Info("Building conversation query")
	queryConversation := query.BuildConversationQuery()

	start := time.Now() // Marca el inicio justo después de ingresar datos

	// Obtener todas las conversaciones paginadas
	logger.Log.Info("Starting paginated query")
	results, err := query.GetAllConversationsResults(analyticsApi, queryConversation)
	if err != nil {
		logger.Log.Error("Error during paginated query", zap.Error(err))
		return
	}
	logger.Log.Info("Successfully retrieved conversation data", zap.Int("TotalConversations", len(results)))

	// Extraer IDs de conversación
	var conversationIDs []string
	for _, conv := range results {
		if conv.ConversationId != nil {
			conversationIDs = append(conversationIDs, *conv.ConversationId)
		}
	}

	// Construir solicitud batch
	const batchWorkers = 5 // O el número que quieras probar

	batchRequestBody, err := functions.AddConversationRecordingsToBatch(conversationIDs, recordApi, batchWorkers)
	if err != nil {
		logger.Log.Warn("Continuing despite some metadata fetch errors", zap.Error(err))
		// No retornamos. Continuamos mientras tengamos algo que procesar.
	}

	if batchRequestBody == nil || len(batchRequestBody) == 0 {
		logger.Log.Warn("No batch requests were created. Exiting.")
		return
	}

	// Enviar batch dividido en partes
	batchSubmissionResults, err := functions.SendBatchRequests(recordApi, batchRequestBody)
	if err != nil {
		logger.Log.Error("Error sending batch requests", zap.Error(err))
		return
	}
	if batchSubmissionResults == nil || len(batchSubmissionResults) == 0 {
		logger.Log.Warn("Batch submission was empty, skipping download")
		return
	}

	// Hacer polling para cada job y descargar grabaciones
	// Procesar jobs en paralelo
	var wg sync.WaitGroup
	maxDownloadWorkers := cfg.MaxDownloadWorkers //10
	pollRetries := cfg.PollRetries               // 100 intentos para evitar timeout
	pollInterval := cfg.PollInterval             //25 * time.Second // tiempo entre polls

	for _, batchSubmissionResult := range batchSubmissionResults {
		jobID := *batchSubmissionResult.Id
		wg.Add(1)

		go func(jobID string) {
			defer wg.Done()

			batchStatus, err := functions.PollBatchJobUntilReady(recordApi, jobID, pollRetries, pollInterval)
			if err != nil {
				logger.Log.Error("Error polling batch job status", zap.String("BatchID", jobID), zap.Error(err))
				return
			}
			if batchStatus == nil || batchStatus.Results == nil {
				logger.Log.Warn("No results available in batch job", zap.String("BatchID", jobID))
				return
			}

			logger.Log.Info("Descargando grabaciones del batch", zap.String("BatchID", jobID))

			err = functions.DownloadAllReadyRecordings(batchStatus, downloadPath, maxDownloadWorkers)
			if err != nil {
				logger.Log.Error("Error downloading recordings", zap.String("BatchID", jobID), zap.Error(err))
			}
		}(jobID)
	}

	wg.Wait()

	elapsed := time.Since(start)
	logger.Log.Info("Tiempo total de ejecución", zap.Duration("Duración:", elapsed))
	logger.Log.Info("Process completed successfully")
}
