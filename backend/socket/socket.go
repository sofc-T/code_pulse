package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sofc-t/code_pulse/delivery/controller"
	"github.com/sofc-t/code_pulse/models"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	documentWebSockets = make(map[string]*DocumentWebSocket)
	documentCache sync.Map
)
type DocumentWebSocket struct {
	Connections map[*websocket.Conn]bool
	Mutex       sync.Mutex
}

func HandleWebSocket(ctx *gin.Context, documentID string) {
	fmt.Println("Handling WebSocket connection for document:", documentID)
	fmt.Println("Connection handled by server running on port:", os.Getenv("PORT"))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		// Allow any origin (not recommended for production, consider a more restrictive check)
		return true
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Get or create a WebSocket instance for the documentID
	documentWebSocket, ok := documentWebSockets[documentID]
	if !ok {
		documentWebSocket = &DocumentWebSocket{Connections: make(map[*websocket.Conn]bool)}
		documentWebSockets[documentID] = documentWebSocket
	}
	fmt.Println("Number of active connections:", len(documentWebSockets[documentID].Connections)+1)

	// Add the new connection to the WebSocket instance
	documentWebSocket.Mutex.Lock()
	documentWebSocket.Connections[conn] = true
	documentWebSocket.Mutex.Unlock()

	// Create a channel to signal when a client disconnects
	disconnectChannel := make(chan *websocket.Conn, 1)

	// Save the source connection
	sourceConnection := conn

	// Start a goroutine to handle disconnection cleanup
	go func() {
		<-disconnectChannel
		// Remove the connection from the WebSocket instance when the client disconnects
		documentWebSocket.Mutex.Lock()
		delete(documentWebSocket.Connections, sourceConnection)
		if len(documentWebSocket.Connections) == 0 {
			fmt.Println("No more connections. Cleaning up resources for document:", documentID)
			documentCache.Delete(documentID)
			delete(documentWebSockets, documentID)
		}
		documentWebSocket.Mutex.Unlock()
	}()

	// Save the source connection
	// sourceConnection := conn

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		var message models.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Error unmarshalling document:", err)
			continue
		}

		// Update the document cache
		if err := UpdateDocumentCache(documentID, message.Data); err != nil {
			log.Println("Error updating document cache:", err)
			continue
		}

		// Update the document in the database
		// if err := documentController.UpdateDocument(documentID, message.Data); err != nil {
		// 	log.Println("Error updating document in DB:", err)
		// 	continue
		// }

		// Broadcast the message to all connected clients for the document
		documentWebSocket.Mutex.Lock()
		for conn := range documentWebSocket.Connections {
			// Skip broadcasting to the source connection
			if conn == sourceConnection {
				continue
			}

			data, err := json.Marshal(message.Change)
			if err != nil {
				log.Println("Error marshalling message:", err)
				continue
			}

			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Println("Error writing message:", err)
				conn.Close()
				disconnectChannel <- conn // Signal disconnection to the cleanup goroutine
				delete(documentWebSocket.Connections, conn)
			}
		}
		documentWebSocket.Mutex.Unlock()
	}
	close(disconnectChannel)
}


func InitializeDocumentCache(ctx *gin.Context, documentController controller.DocumentController) (*models.Document, error) {
	documentID := ctx.Param("id")
	var document *models.Document

	// Check if the document is already in the cache
	if cachedDocument, ok := documentCache.Load(documentID); !ok {
		// Fetch the document from the database
		document, err := documentController.GetOneDocument(ctx)
		if err != nil {
			fmt.Println("Error getting document:", err)
			return nil, err
		}

		// Store the document in the cache
		documentCache.Store(documentID, document)
		ctx.JSON(http.StatusOK, document)
	} else {
		// If document is already in cache, retrieve it
		document = cachedDocument.(*models.Document)
	}
	return document, nil
}

func UpdateDocumentCache(documentID string, newData models.DocumentData) error {
	cachedDocument, ok := documentCache.Load(documentID)
	if !ok {
		return fmt.Errorf("document not found in cache")
	}

	document := cachedDocument.(*models.Document)
	document.Data = newData

	// Update the document in the cache
	documentCache.Store(documentID, document)
	return nil
}

func UpdateDatabaseWithCache(documentController controller.DocumentController) {
	// Create a ticker that ticks every specified duration
	ticker := time.NewTicker(30 * time.Second)

	// Run a goroutine to perform the periodic update
	go func() {
		for range ticker.C {
			// Perform the database update using the cache
			err := syncDatabaseWithCache(documentController)
			if err != nil {
				fmt.Println("Error updating database with cache:", err)
			}
		}
	}()
}

func syncDatabaseWithCache(documentController controller.DocumentController) error {
	// Iterate over the cache and update the database with each entry
	documentCache.Range(func(key, value interface{}) bool {
		documentID := key.(string)
		cachedDocument := value.(*models.Document)

		// Update the database with the cached document
		err := documentController.UpdateDocument(documentID, cachedDocument.Data)
		if err != nil {
			// Log or handle the error accordingly
			fmt.Printf("Error updating database for document %s: %v\n", documentID, err)
		}
		// documentCache = sync.Map{}
		return true
	})

	return nil
}

func UpdateDocumentCacheAttribute(documentID string, newData models.Access) error {
	fmt.Print("Updating document cache attribute\n")
	cachedDocument, ok := documentCache.Load(documentID)
	if !ok {
		return fmt.Errorf("document not found in cache")
	}

	document := cachedDocument.(*models.Document)
	document.ReadAccess = newData.ReadAccess
	document.WriteAccess = newData.WriteAccess

	// Update the document in the cache
	documentCache.Store(documentID, document)
	return nil
}

func UpdateDocumentTitleCacheAttribute(documentID string, newTitle string) error {
	fmt.Print("Updating document title cache attribute\n", documentID, newTitle)
	cachedDocument, ok := documentCache.Load(documentID)
	if !ok {
		return fmt.Errorf("document not found in cache")
	}

	document := cachedDocument.(*models.Document)
	document.Title = newTitle

	// Update the document in the cache
	documentCache.Store(documentID, document)
	return nil
}
