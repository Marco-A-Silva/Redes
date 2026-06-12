<h1 align="center">🌐 Protocolo SCH (Simple Connection Handling)</h1>

<p align="center">
  <em>Implementación personalizada de un protocolo de port forwarding basado en Reverse Proxy.</em>
</p>

## 🎯 Objetivo
Desarrollar un protocolo de comunicación propio (`SCH`) capaz de gestionar *port forwarding* entre servicios. El proyecto simula un entorno de red privada mediante contenedores, permitiendo la comunicación segura y controlada entre un cliente y servicios internos a través de un proxy inverso.

## 🏗 Arquitectura del Proyecto
La arquitectura está diseñada para separar la interfaz de usuario de la lógica de networking y la infraestructura de red.

### Estructura de Componentes
*   **`client/`**: Contiene la interfaz de línea de comandos (CLI) escrita en Go (`main.go`, `ui.go`). Es el punto de entrada para que el usuario interactúe con el túnel.
*   **`proyecto/`**: Núcleo de la infraestructura:
    *   `sch.go`: Implementación del protocolo `SCH` (lógica de serialización/deserialización y manejo de paquetes).
    *   `proxy.go`: Servidor de proxy inverso que actúa como mediador.
    *   `forumDest.go`: Servicio de destino simulado para verificar la conectividad.
*   **Entorno (Docker)**: El archivo `docker-compose.yml` (visible en `image_92ddbb.png`) orquesta los contenedores, creando una subred aislada donde se despliegan el proxy y los servicios, emulando una topología de red real.

## 🛠 Tecnologías y Stack
*   **Lenguaje:** Go (Golang) - *Aprovechando su capacidad de concurrencia para el manejo de streams de red.*
*   **Infraestructura:** Docker & Docker Compose.
*   **Networking:** Implementación de protocolos a nivel de aplicación (Port Forwarding / Proxy).

## 🚀 Cómo probar el sistema
1. **Levantar la red:**
   Asegúrate de tener Docker instalado y ejecuta el entorno simulado:
```bash
   docker-compose up --build
```
2. **Ejecutar el cliente:**
   Compila y corre el cliente para iniciar la conexión:
```bash
   go run client/main.go
```
