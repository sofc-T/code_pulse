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
	"github.com/streadway/amqp" 
	"github.com/sofc-t/code_pulse/delivery/controller"
	"github.com/sofc-t/code_pulse/models"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	documentWebSockets = make(map[string]*DocumentWebSocket)
	documentCache      sync.Map
	rabbitConn         *amqp.Connection
	rabbitChannel      *amqp.Channel
	exchangeName       = "document_updates"
)

type DocumentWebSocket struct {
	Connections map[*websocket.Conn]bool
	Mutex       sync.Mutex
}

func InitRabbitMQ() error {
	var err error
	rabbitConn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	rabbitChannel, err = rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	err = rabbitChannel.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	fmt.Println("RabbitMQ initialized successfully")
	return nil
}


func PublishMessage(documentID string, message models.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = rabbitChannel.Publish(
		exchangeName, // exchange
		"",           // routing key (ignored in fanout)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Println("Message published to RabbitMQ")
	return nil
}


func StartRabbitMQConsumer() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	queue, err := channel.QueueDeclare(
		"",    // auto-generated queue name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = channel.QueueBind(
		queue.Name,
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	msgs, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	go func() {
		for msg := range msgs {
			var message models.Message
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Println("Failed to unmarshal message:", err)
				continue
			}
			BroadcastMessage(message)
		}
	}()

	fmt.Println("RabbitMQ consumer started")
}

func BroadcastMessage(message models.Message) {
	documentWebSocket, ok := documentWebSockets[message.ID]
	if !ok {
		return
	}

	documentWebSocket.Mutex.Lock()
	defer documentWebSocket.Mutex.Unlock()

	data, err := json.Marshal(message.Change)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	for conn := range documentWebSocket.Connections {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("Error writing message:", err)
			conn.Close()
			delete(documentWebSocket.Connections, conn)
		}
	}
}

// Handle incoming WebSocket connections
func HandleWebSocket(ctx *gin.Context, documentID string) {
	fmt.Println("Handling WebSocket connection for document:", documentID)
	fmt.Println("Connection handled by server running on port:", os.Getenv("PORT"))

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	documentWebSocket, ok := documentWebSockets[documentID]
	if !ok {
		documentWebSocket = &DocumentWebSocket{Connections: make(map[*websocket.Conn]bool)}
		documentWebSockets[documentID] = documentWebSocket
	}

	documentWebSocket.Mutex.Lock()
	documentWebSocket.Connections[conn] = true
	documentWebSocket.Mutex.Unlock()

	disconnectChannel := make(chan *websocket.Conn, 1)
	sourceConnection := conn

	go func() {
		<-disconnectChannel
		documentWebSocket.Mutex.Lock()
		delete(documentWebSocket.Connections, sourceConnection)
		if len(documentWebSocket.Connections) == 0 {
			fmt.Println("No more connections. Cleaning up resources for document:", documentID)
			documentCache.Delete(documentID)
			delete(documentWebSockets, documentID)
		}
		documentWebSocket.Mutex.Unlock()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		var message models.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Error unmarshalling document:", err)
			continue
		}

		if err := UpdateDocumentCache(documentID, message.Data); err != nil {
			log.Println("Error updating document cache:", err)
			continue
		}

		
		err = PublishMessage(documentID, message)
		if err != nil {
			log.Println("Error publishing to RabbitMQ:", err)
		}
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
	ticker := time.NewTicker(10 * time.Second)

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
