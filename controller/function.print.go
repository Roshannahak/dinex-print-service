package controller

import (
	"dinex-print-service/model"
	"fmt"
	"strings"
	"time"
)

// 58mm
func GenerateThermalBill58mm(inv  model.BillDetail, restro model.RestaurantDetail) string {
	return generateThermalBill(inv, restro, 28, "----------------------------\n")
}

// 80mm
func GenerateThermalBill80mm(inv model.BillDetail, restro model.RestaurantDetail) string {
	return generateThermalBill(inv, restro, 32, "--------------------------------\n")
}

// 112mm
func GenerateThermalBill112mm(inv model.BillDetail, restro model.RestaurantDetail) string {
	return generateThermalBill(inv, restro, 48, "------------------------------------------------\n")
}
func generateThermalBill(inv model.BillDetail, restro model.RestaurantDetail, width int, line string) string {

	// date formatter
	t := time.Unix(inv.CreatedAt, 0)
	formattedTime := t.Format("02-01-2006 03:04 PM")

	var output string

	output += centerText(restro.Name, width)
	output += centerText(restro.Address+","+restro.City, width)
	output += centerText("GSTIN: "+inv.Pricing.Tax.GSTIN, width)
	output += line

	output += fmt.Sprintf("Invoice: %s\n", inv.InvoiceNo)
	output += fmt.Sprintf("Date: %s\n", formattedTime)

	if inv.OrderType == "DINE_IN" {
		output += fmt.Sprintf("Table: %s\n", inv.Table.TableNo)
	}

	output += line

	// Dynamic trim based on width
	nameLimit := width - 12

	for _, item := range inv.Items {
		name := trimText(item.Name, nameLimit)
		qtyPrice := fmt.Sprintf("%d x %.2f", item.Quantity, item.Price)
		output += formatTwoColumn(name, qtyPrice, width)
	}

	output += line

	output += formatTwoColumn("Subtotal:", fmt.Sprintf("%.2f", inv.Pricing.Subtotal), width)
	output += formatTwoColumn("CGST:", fmt.Sprintf("%.2f", inv.Pricing.Tax.CGST), width)
	output += formatTwoColumn("SGST:", fmt.Sprintf("%.2f", inv.Pricing.Tax.SGST), width)
	output += line
	output += formatTwoColumn("TOTAL:", fmt.Sprintf("%.2f", inv.Pricing.GrandTotal), width)
	output += line

	if len(inv.Payment) > 0 {
		for _, p := range inv.Payment {
			output += fmt.Sprintf("Payment (%s): %.2f\n", p.Method, p.Amount)
		}
	} else {
		output += "Payment: N/A\n"
	}

	output += "\nThank You! Visit Again\n\n\n"

	return output
}
func centerText(text string, width int) string {
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + "\n"
}

func formatTwoColumn(left, right string, width int) string {
	space := width - len(left) - len(right)
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right + "\n"
}

func trimText(text string, max int) string {
	if len(text) > max {
		return text[:max]
	}
	return text
}

// check upi avalable on bill or not
func hasUPIPayment(payments []model.PaymentDetail) (bool, float64) {
	for _, p := range payments {
		if strings.ToUpper(p.Method) == "UPI" {
			return true, p.Amount
		}
	}
	return false, 0
}

// 58mm
func GenerateThermalKot58mm(kds model.Kds) string {
	return generateThermalKot(kds, 28, "----------------------------\n")
}

// 80mm
func GenerateThermalKot80mm(kds model.Kds) string {
	return generateThermalKot(kds, 32, "--------------------------------\n")
}

// 112
func GenerateThermalKot112mm(kds model.Kds) string {
	return generateThermalKot(kds, 48, "------------------------------------------------\n")
}

func generateThermalKot(kds model.Kds, width int, line string) string {

	// Time format
	loc, err := time.LoadLocation("Asia/Kolkata")
	var timestamp time.Time
	if err != nil {
		timestamp = time.Unix(kds.CreatedAt, 0).Local()
	} else {
		timestamp = time.Unix(kds.CreatedAt, 0).In(loc)
	}
	formattedTime := timestamp.Format("02-01-2006 03:04 PM")

	var output string

	// Header
	output += centerText(kds.KdsNumber, width)
	output += line

	// Basic Info
	output += fmt.Sprintf("order: %s-%s\n", kds.OrderId, kds.TableNumber)
	output += fmt.Sprintf("time: %s\n", formattedTime)

	output += line

	// Items (only trim adjusted based on width)
	nameLimit := width - 10

	for _, item := range kds.KdsItems {
		itemName := item.Name + "(" + string(item.FoodCategory) + ")"
		name := trimText(itemName, nameLimit)
		qty := fmt.Sprintf("x%d", item.Quantity)
		output += formatTwoColumn(name, qty, width)
	}

	output += line
	output += "\n\n\n"

	return output
}
