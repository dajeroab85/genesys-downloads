package functions

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goDownloadRecording/logger"
	sdk "github.com/mypurecloud/platform-client-sdk-go/v157/platformclientv2"
	"go.uber.org/zap"
)

// safeString evita valores nulos
func safeString(field *string) string {
	if field != nil {
		return *field
	}
	return "N/A"
}

// writeMetadataFile guarda los metadatos en metadata.txt
func writeMetadataFile(folderPath string, item sdk.Batchdownloadjobresult) error {
	metadataPath := filepath.Join(folderPath, "metadata.txt")
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "Recording ID: %s\nConversation ID: %s\nContent Type: %s\nResult URL: %s\nFecha de descarga: %s\n",
		safeString(item.RecordingId),
		safeString(item.ConversationId),
		safeString(item.ContentType),
		safeString(item.ResultUrl),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	return err
}

// downloadFile descarga un archivo desde una URL al path local
func downloadFile(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: status code %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// DownloadAllReadyRecordings descarga las grabaciones en paralelo, cada una en su carpeta
func DownloadAllReadyRecordings(result *sdk.Batchdownloadjobstatusresult, outputDir string, maxWorkers int) error {
	if result == nil || result.Results == nil {
		logger.Log.Warn("No results to download")
		return nil
	}

	start := time.Now()
	tasks := make(chan sdk.Batchdownloadjobresult, len(*result.Results))
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for item := range tasks {
				if item.ResultUrl == nil || item.RecordingId == nil || item.ConversationId == nil {
					logger.Log.Warn("Missing ResultUrl, RecordingId or ConversationId, skipping item")
					continue
				}

				// Crear carpeta con formato YYMMDD-ConversationId
				folderName := time.Now().Format("060102") + "-" + safeString(item.ConversationId)
				folderPath := filepath.Join(outputDir, folderName)
				os.MkdirAll(folderPath, os.ModePerm)

				// Descargar grabación
				fileName := safeString(item.RecordingId) + ".mp3"
				filePath := filepath.Join(folderPath, fileName)
				err := downloadFile(*item.ResultUrl, filePath)
				if err != nil {
					logger.Log.Error("Failed to download recording", zap.String("RecordingID", *item.RecordingId), zap.Error(err))
					continue
				}

				logger.Log.Info("Downloaded recording",
					zap.String("File", filePath),
					zap.Int("Worker", workerID),
				)

				// Guardar metadata.txt
				err = writeMetadataFile(folderPath, item)
				if err != nil {
					logger.Log.Error("Failed to write metadata", zap.String("RecordingID", *item.RecordingId), zap.Error(err))
				}
			}
		}(i)
	}

	// Enviar tareas
	for _, item := range *result.Results {
		tasks <- item
	}
	close(tasks)
	wg.Wait()

	elapsed := time.Since(start)
	logger.Log.Info("All downloads completed",
		zap.Int("TotalFiles", len(*result.Results)),
		zap.Int("Workers", maxWorkers),
		zap.Duration("Duration", elapsed),
	)
	return nil
}
