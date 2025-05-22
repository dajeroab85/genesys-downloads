# Genesys Cloud Recordings Downloader

Aplicación en Go para descargar grabaciones de Genesys Cloud en paralelo, organizándolas en carpetas por fecha y ConversationID, con manejo eficiente de concurrencia y registro detallado de logs.

---

## Características

    - Descarga masiva de grabaciones desde Genesys Cloud usando su SDK oficial en Go.
    - Descarga concurrente con un número configurable de trabajadores (`MAX_DOWNLOAD_WORKERS`).
    - Organización de grabaciones en carpetas con formato `yymmdd-conversationId`.
    - Generación de un archivo de metadata `.txt` por grabación con detalles adicionales.
    - Configuración flexible mediante archivo `.env`.
    - Registro detallado de logs para monitoreo y depuración.

---

## Requisitos

    - Go 1.20 o superior
    - Cuenta y credenciales de Genesys Cloud (Client ID y Client Secret)
    - Acceso a la API de Genesys Cloud con permisos adecuados
    - [Go Modules](https://github.com/golang/go/wiki/Modules) habilitado

---

## Instalación

1. Clonar el repositorio

   ```bash
   git clone https://github.com/tu-usuario/genesys-recordings-downloader.git
   cd genesys-recordings-downloader


2. Instalar Dependencias

    ```bash
    go mod tidy


3. Configurar credenciales y entorno

    Debes Crear el archivo .env con las variables de entorno para configurar:

    GENESYS_ENVIRONMENT= environment
    CLIENT_ID= clientId
    CLIENT_SECRET= clientSecret
    MAX_DOWNLOAD_WORKERS=15
    POLL_RETRIES=50
    POLL_INTERVAL=25

## Uso

    Ejecuta la aplicación:
        ```bash
        go run main.go

    La aplicación descargará las grabaciones dentro de la carpeta configurada, creando subcarpetas con el formato yymmdd-conversationId.


## Personalización

    Número de trabajadores: Modifica MAX_DOWNLOAD_WORKERS en el archivo .env para controlar la cantidad de descargas simultáneas.

    Rango de fechas: Puedes ajustar el rango de fechas desde el código que realiza la consulta a la API de Genesys.

## Estructura del Proyecto

.
├── config/        # Configuración y carga de variables de entorno

├── functions/     # Funciones para descarga, procesamiento y escritura

├── logger/        # Configuración del logger con zap

├── logs/          # Carpeta donde se escriben los logs

├── main.go        # Punto de entrada de la aplicación

├── .env           # Variables de entorno (no incluir en Git)

└── .gitignore     # Archivos ignorados por Git



## Contribuciones

¡Contribuciones bienvenidas!
Por favor, abre un issue o un pull request para sugerencias, mejoras o correcciones.

## Licencia

Este proyecto está bajo la licencia MIT.

Contacto
Daniel J Rodríguez A