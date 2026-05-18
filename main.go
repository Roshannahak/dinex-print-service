package main

import (
	"bufio"
	"bytes"
	"dinex-print-service/controller"
	"dinex-print-service/model"
	"encoding/base64"
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

		size := c.DefaultQuery("size", "58mm")

		content := "DineX TEST PRINT\n----------------\nDate: " + time.Now().Format("2006-01-02 15:04:05") + "\nSize: " + size + "\nStatus: Working\n----------------\nThank you!"

		printerName, printType, err := AutoDetectPrinter(size, "")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "No printer detected"})
			return
		}

		receipt := GenerateNormalText(content)

		switch printType {
		case "USB":
			err = PrintLaser(printerName, string(receipt), "", size)
		case "BLUETOOTH":
			err = PrintBluetooth(printerName, receipt)
		case "LAN":
			err = PrintLAN(printerName, receipt)
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

	r.POST("/printbill", func(c *gin.Context) {

		var req model.PrintBillRequest

		if err := c.ShouldBindJSON(&req); err != nil {

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
			err := AutoDetectPrinter(req.PrintSize, req.PrinterIP)

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

		var receiptContent string
		switch req.PrintSize {
		case "80mm":
			receiptContent = controller.GenerateThermalBill80mm(req.Bill, req.Restaurant)
		case "112mm":
			receiptContent = controller.GenerateThermalBill112mm(req.Bill, req.Restaurant)
		default:
			receiptContent = controller.GenerateThermalBill58mm(req.Bill, req.Restaurant)
		}

		receipt := []byte(receiptContent)

		//------------------------------------------------
		// HANDLE QR
		//------------------------------------------------
		//------------------------------------------------
		// HANDLE QR (Use high-quality graphical printing for all formats)
		//------------------------------------------------
		if req.QR != "" {
			err = PrintLaser(printerName, receiptContent, req.QR, req.PrintSize)
			if err != nil {
				c.JSON(500, gin.H{"success": false, "message": err.Error()})
				return
			}
			c.JSON(200, model.PrintResponse{
				Success:   true,
				Message:   "printed successfully",
				Printer:   printerName,
				PrintType: printType,
				PrintedAt: time.Now().Format(time.RFC3339),
			})
			return
		}

		//------------------------------------------------
		// PRINT
		//------------------------------------------------

		switch printType {
		case "USB":
			err = PrintLaser(printerName, receiptContent, "", req.PrintSize)
		case "BLUETOOTH":
			err = PrintBluetooth(printerName, receipt)
		case "LAN":
			err = PrintLAN(printerName, receipt)
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

		c.JSON(200, model.PrintResponse{
			Success:   true,
			Message:   "printed successfully",
			Printer:   printerName,
			PrintType: printType,
			PrintedAt: time.Now().Format(time.RFC3339),
		})
	})

	//------------------------------------------------
	// PRINT KOT
	//------------------------------------------------

	r.POST("/printkot", func(c *gin.Context) {

		var req model.PrintKotRequest

		if err := c.ShouldBindJSON(&req); err != nil {

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
			err := AutoDetectPrinter(req.PrintSize, req.PrinterIP)

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

		var receiptContent string
		switch req.PrintSize {
		case "80mm":
			receiptContent = controller.GenerateThermalKot80mm(req.Kot)
		case "112mm":
			receiptContent = controller.GenerateThermalKot112mm(req.Kot)
		default:
			receiptContent = controller.GenerateThermalKot58mm(req.Kot)
		}

		receipt := []byte(receiptContent)

		//------------------------------------------------
		// PRINT
		//------------------------------------------------

		switch printType {
		case "USB":
			err = PrintLaser(printerName, receiptContent, "", req.PrintSize)
		case "BLUETOOTH":
			err = PrintBluetooth(printerName, receipt)
		case "LAN":
			err = PrintLAN(printerName, receipt)
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

		c.JSON(200, model.PrintResponse{
			Success:   true,
			Message:   "printed successfully",
			Printer:   printerName,
			PrintType: printType,
			PrintedAt: time.Now().Format(time.RFC3339),
		})
	})

	log.Println("DineX Print Service Running :3232")

	r.Run(":3232")
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
	paperSize string,
	printerIP string,
) (string, string, error) {

	//------------------------------------------------
	// LAN
	//------------------------------------------------

	if printerIP != "" {

		return printerIP,
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
				DetectPrinterType(p),
				nil
		}
	}

	return names[0],
		DetectPrinterType(names[0]),
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

	err = p.StartDocument("DineX Receipt", "TEXT")

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
// LASER PRINT (WITH GRAPHICS)
// ----------------------------------------------------

func PrintLaser(printerName string, text string, qrBase64 string, paperSize string) error {

	// 1. Save QR to temp file
	qrPath := os.TempDir() + "\\dinex_qr.png"
	qrBytes, err := base64.StdEncoding.DecodeString(qrBase64)
	if err == nil {
		os.WriteFile(qrPath, qrBytes, 0644)
	}

	// 2. Escape text for PowerShell
	escapedText := strings.ReplaceAll(text, "`", "``")
	escapedText = strings.ReplaceAll(escapedText, "\"", "`\"")
	escapedText = strings.ReplaceAll(escapedText, "$", "`$")

	// 3. Build PowerShell Script (Use [char]10 to avoid backtick conflict in Go raw strings)
	psScript := fmt.Sprintf(`
		Add-Type -AssemblyName System.Drawing
		$pd = New-Object System.Drawing.Printing.PrintDocument
		$pd.PrinterSettings.PrinterName = "%s"
		$pd.add_PrintPage({
			$g = $_.Graphics
			
			$fontSize = 9.0
			$paperWidth = 200 # 58mm Default
			$qrWidth = 80
			
			if ("%s" -eq "80mm") { $paperWidth = 300; $qrWidth = 110; $fontSize = 11.5 }
			if ("%s" -eq "112mm") { $paperWidth = 420; $qrWidth = 140; $fontSize = 11.0 }
			
			$f = New-Object System.Drawing.Font("Courier New", [float]$fontSize)
			$y = 10

			# Use the smaller of detected width or our defined width
			$actualWidth = [Math]::Min($_.PageBounds.Width, $paperWidth)

			# Print Main Bill Text
			$text = @"
%s
"@
			$g.DrawString($text, $f, [System.Drawing.Brushes]::Black, 10, $y)
			
			# Calculate Y position after text
			$textSize = $g.MeasureString($text, $f)
			$y += $textSize.Height - 2

			if ("%s" -ne "") {
				# Print "Scan & Pay" Label
				$headerFont = New-Object System.Drawing.Font("Courier New", 10, [System.Drawing.FontStyle]::Bold)
				$headerText = "Scan & Pay"
				$headerSize = $g.MeasureString($headerText, $headerFont)
				$headerX = ($actualWidth - $headerSize.Width) / 2
				$g.DrawString($headerText, $headerFont, [System.Drawing.Brushes]::Black, $headerX, $y)
				$y += $headerSize.Height

				# Print QR Image (Centered)
				if (Test-Path "%s") {
					$img = [System.Drawing.Image]::FromFile("%s")
					$qrX = ($actualWidth - $qrWidth) / 2
					$g.DrawImage($img, $qrX, $y, $qrWidth, $qrWidth)
					$img.Dispose()
				}
			}
		})
		$pd.Print()
	`, printerName, paperSize, paperSize, escapedText, qrBase64, qrPath, qrPath)

	// 4. Execute PowerShell
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell error: %v, output: %s", err, string(output))
	}

	return nil
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
