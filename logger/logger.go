package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	// Definir nivel de logs
	logLevel := zapcore.DebugLevel

	// Configura encoder para consola (desarrollo)
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	// Configura encoder para archivo (JSON estructurado)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// Abre archivo de log con nombre por fecha
	today := time.Now().Format("2006-01-02")
	logFile, err := os.OpenFile("logs/"+today+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("No se pudo crear el archivo de log: " + err.Error())
	}

	// Crear escritor para archivo
	fileWriter := zapcore.AddSync(logFile)
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Combine ambos escritores
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriter, logLevel),       // archivo
		zapcore.NewCore(consoleEncoder, consoleWriter, logLevel), // consola
	)

	// Inicializa el logger
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
