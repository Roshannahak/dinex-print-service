package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alexbrainman/printer"
	"github.com/gin-gonic/gin"
)

type PrintJob struct {
	Type       string `json:"type"`
	PaperSize  string `json:"paper_size"` // 58mm,80mm,112mm
	Content    string `json:"content"`
	PrinterIP  string `json:"printer_ip,omitempty"`
	Copies     int    `json:"copies"`
	Restaurant string `json:"restaurant"`
}

type PrintResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Printer   string `json:"printer"`
	PrintType string `json:"print_type"`
	PrintedAt string `json:"printed_at"`
}

func main() {

	r := gin.Default()

	r.Use(CORSMiddleware())

	//------------------------------------------------
	// HEALTH
	//------------------------------------------------

	r.GET("/health", func(c *gin.Context) {

		c.JSON(200, gin.H{
			"status": "running",
		})
	})

	//------------------------------------------------
	// DETECT PRINTERS
	//------------------------------------------------

	r.GET("/printers", func(c *gin.Context) {

		printers := DetectPrinters()

		c.JSON(200, gin.H{
			"printers": printers,
		})
	})

	//------------------------------------------------
	// TEST PRINT (DUMMY)
	//------------------------------------------------

	r.GET("/test-print", func(c *gin.Context) {

		size := c.DefaultQuery("size", "80mm")

		dummyJob := PrintJob{
			Content:   "DineX TEST PRINT\n----------------\nDate: " + time.Now().Format("2006-01-02 15:04:05") + "\nSize: " + size + "\nStatus: Working\n----------------\nThank you!",
			PaperSize: size,
		}

		printerName, printType, err := AutoDetectPrinter(dummyJob)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "No printer detected"})
			return
		}

		receipt := GenerateNormalText(dummyJob.Content)

		switch printType {
		case "USB":
			err = PrintUSB(printerName, receipt)
		case "BLUETOOTH":
			err = PrintBluetooth(printerName, receipt)
		case "LAN":
			err = PrintLAN(dummyJob.PrinterIP, receipt)
		}

		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test receipt sent to " + printerName,
			"size":    size,
		})
	})

	//------------------------------------------------
	// PRINT
	//------------------------------------------------

	r.POST("/print", func(c *gin.Context) {

		var job PrintJob

		if err := c.ShouldBindJSON(&job); err != nil {

			c.JSON(400, gin.H{
				"success": false,
				"message": err.Error(),
			})

			return
		}

		//------------------------------------------------
		// DETECT PRINTER
		//------------------------------------------------

		printerName,
			printType,
			err := AutoDetectPrinter(job)

		if err != nil {

			c.JSON(500, gin.H{
				"success": false,
				"message": err.Error(),
			})

			return
		}

		//------------------------------------------------
		// GENERATE CONTENT
		//------------------------------------------------

		receipt := GenerateNormalText(
			job.Content,
		)

		//------------------------------------------------
		// PRINT
		//------------------------------------------------

		switch printType {

		case "USB":

			err = PrintUSB(
				printerName,
				receipt,
			)

		case "BLUETOOTH":

			err = PrintBluetooth(
				printerName,
				receipt,
			)

		case "LAN":

			err = PrintLAN(
				job.PrinterIP,
				receipt,
			)
		}

		if err != nil {

			c.JSON(500, gin.H{
				"success": false,
				"message": err.Error(),
			})

			return
		}

		//------------------------------------------------
		// ACKNOWLEDGEMENT
		//------------------------------------------------

		c.JSON(200, PrintResponse{
			Success:   true,
			Message:   "printed successfully",
			Printer:   printerName,
			PrintType: printType,
			PrintedAt: time.Now().Format(time.RFC3339),
		})
	})

	log.Println("DineX Print Service Running :8080")

	r.Run(":8080")
}

// ----------------------------------------------------
// DETECT PRINTERS
// ----------------------------------------------------

func DetectPrinters() []map[string]string {

	var result []map[string]string

	statuses := GetPrinterStatuses()

	//------------------------------------------------
	// USB/WINDOWS PRINTERS
	//------------------------------------------------

	names, err := printer.ReadNames()

	if err == nil {

		for _, p := range names {

			status := statuses[p]

			result = append(result, map[string]string{
				"name":   p,
				"type":   DetectPrinterType(p),
				"status": TranslateStatus(status),
			})
		}
	}

	return result
}

// ----------------------------------------------------
// GET STATUSES
// ----------------------------------------------------

func GetPrinterStatuses() map[string]string {

	statuses := make(map[string]string)

	cmd := exec.Command("wmic", "printer", "get", "Name,PrinterStatus")
	output, _ := cmd.Output()

	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip header
	if scanner.Scan() {
	}

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {

			status := parts[len(parts)-1]
			name := strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
			statuses[name] = status
		}
	}

	return statuses
}

// ----------------------------------------------------
// TRANSLATE STATUS
// ----------------------------------------------------

func TranslateStatus(code string) string {

	switch code {
	case "1":
		return "Other"
	case "2":
		return "Unknown"
	case "3":
		return "Idle"
	case "4":
		return "Printing"
	case "5":
		return "Warming Up"
	case "6":
		return "Stopped"
	case "7":
		return "Offline"
	case "8":
		return "Paused"
	case "9":
		return "Error"
	default:
		return "Ready"
	}
}

// ----------------------------------------------------
// AUTO DETECT
// ----------------------------------------------------

func AutoDetectPrinter(
	job PrintJob,
) (string, string, error) {

	//------------------------------------------------
	// LAN
	//------------------------------------------------

	if job.PrinterIP != "" {

		return job.PrinterIP,
			"LAN",
			nil
	}

	//------------------------------------------------
	// WINDOWS PRINTERS
	//------------------------------------------------

	names, err := printer.ReadNames()

	if err != nil {

		return "",
			"",
			err
	}

	if len(names) == 0 {

		return "",
			"",
			fmt.Errorf("no printer found")
	}

	//------------------------------------------------
	// PRIORITIZE DEFAULT PRINTER
	//------------------------------------------------

	defaultPrinter, err := printer.Default()
	if err == nil {
		return defaultPrinter, "USB", nil
	}

	//statuses := GetPrinterStatuses()

	//------------------------------------------------
	// FIND ACTIVE/THERMAL PRINTER
	//------------------------------------------------

	for _, p := range names {

		upper := strings.ToUpper(p)

		if strings.Contains(upper, "POS") ||
			strings.Contains(upper, "58") ||
			strings.Contains(upper, "80") ||
			strings.Contains(upper, "XP-") || // Changed from "XP" to avoid "XPS"
			strings.Contains(upper, "THERMAL") ||
			strings.Contains(upper, "EPSON") ||
			strings.Contains(upper, "HP") || // Added for user's printer
			strings.Contains(upper, "LASERJET") ||
			strings.Contains(upper, "MFP") {

			return p,
				"USB",
				nil
		}
	}

	return names[0],
		"USB",
		nil
}

// ----------------------------------------------------
// DETECT TYPE
// ----------------------------------------------------

func DetectPrinterType(
	name string,
) string {

	upper := strings.ToUpper(name)

	if strings.Contains(upper, "BLUETOOTH") {

		return "BLUETOOTH"
	}

	if strings.Contains(upper, "TCP") ||
		strings.Contains(upper, "LAN") ||
		strings.Contains(upper, "NETWORK") {

		return "LAN"
	}

	return "USB"
}

// ----------------------------------------------------
// NORMAL TEXT
// ----------------------------------------------------

func GenerateNormalText(
	content string,
) []byte {

	return []byte(content + "\n\n\n")
}

// ----------------------------------------------------
// USB PRINT
// ----------------------------------------------------

func PrintUSB(
	printerName string,
	data []byte,
) error {

	p, err := printer.Open(
		printerName,
	)

	if err != nil {

		return err
	}

	defer p.Close()

	err = p.StartDocument(
		"DineX Receipt",
		"TEXT",
	)

	if err != nil {

		return err
	}

	defer p.EndDocument()

	err = p.StartPage()

	if err != nil {

		return err
	}

	defer p.EndPage()

	_, err = p.Write(data)

	return err
}

// ----------------------------------------------------
// BLUETOOTH PRINT
// ----------------------------------------------------

func PrintBluetooth(
	port string,
	data []byte,
) error {

	file, err := os.OpenFile(
		port,
		os.O_RDWR,
		0666,
	)

	if err != nil {

		return err
	}

	defer file.Close()

	_, err = file.Write(data)

	return err
}

// ----------------------------------------------------
// LAN PRINT
// ----------------------------------------------------

func PrintLAN(
	ip string,
	data []byte,
) error {

	conn, err := net.DialTimeout(
		"tcp",
		ip+":9100",
		5*time.Second,
	)

	if err != nil {

		return err
	}

	defer conn.Close()

	_, err = conn.Write(data)

	return err
}

// ----------------------------------------------------
// CORS
// ----------------------------------------------------

func CORSMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		c.Writer.Header().Set(
			"Access-Control-Allow-Origin",
			"*",
		)

		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type",
		)

		c.Writer.Header().Set(
			"Access-Control-Allow-Methods",
			"POST, GET, OPTIONS",
		)

		if c.Request.Method == "OPTIONS" {

			c.AbortWithStatus(200)

			return
		}

		c.Next()
	}
}
