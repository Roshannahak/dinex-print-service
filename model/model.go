package model

type PrintBillRequest struct {
	Bill       BillDetail `json:"bill"`
	Restaurant RestaurantDetail `json:"restaurant"`
	PrintSize  string     `json:"printsize"` // 58mm,80mm,112mm
	PrinterIP  string     `json:"printer_ip,omitempty"`
	QR         string     `json:"qr"`
}

type BillDetail struct {
	InvoiceNo string          `json:"invoiceno"`
	OrderType string          `json:"ordertype"`
	Items     []BillItem      `json:"items"`
	Pricing   Pricing         `json:"pricing"`
	Table     TableDetail     `json:"table"`
	Payment   []PaymentDetail `json:"payment"`
	CreatedAt int64           `json:"createdat"`
}

type TableDetail struct {
	ID              string `json:"id"`
	RestaurantObjID string `json:"restaurantobjid"`
	RestaurantID    string `json:"restaurantid"`
	TableNo         string `json:"tableno"`
	Status          string `json:"status"`
	Capacity        int    `json:"capacity"`
	CurrentOrderID  string `json:"currentorderid"`
}

type PaymentDetail struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}

type BillItem struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Total    float64 `json:"itemtotal"`
}

type Pricing struct {
	Subtotal        float64 `json:"subtotal"`
	Discount        float64 `json:"discount"`
	Tax             TaxInfo `json:"tax"`
	PackagingCharge float64 `json:"packagingcharge"`
	RoundOff        float64 `json:"roundoff"`
	GrandTotal      float64 `json:"grandtotal"`
}

type TaxInfo struct {
	GSTIN string  `json:"gstin"`
	CGST  float64 `json:"cgst"`
	SGST  float64 `json:"sgst"`
	Total float64 `json:"total"`
}

type RestaurantDetail struct {
	Name    string `json:"restaurantname"`
	Address string `json:"address"`
	City    string `json:"city"`
	Pincode string `json:"pincode"`
	PhoneNo string `json:"phoneno"`
}

type PrintResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Printer   string `json:"printer"`
	PrintType string `json:"print_type"`
	PrintedAt string `json:"printed_at"`
}

type PrintKotRequest struct {
	Kot       Kds    `json:"kot"`
	PrintSize string `json:"printsize"`
	PrinterIP string `json:"printer_ip,omitempty"`
}

type Kds struct {
	ID              string    `json:"id"`
	OrderId         string    `json:"orderid"`
	RestaurantObjId string    `json:"restaurantobjid"`
	KdsNumber       string    `json:"kdsnumber"`
	TableNumber     string    `json:"table"` // maps to "table" in JSON
	KdsItems        []KdsItem `json:"kdsitems"`
	Printed         bool      `json:"printed"`
	Status          string    `json:"status"`
	CreatedAt       int64     `json:"createdat"`
	UpdatedAt       int64     `json:"updatedat"`
}

type KdsItem struct {
	ItemId       string `json:"itemid"`
	Name         string `json:"name"`
	FoodCategory string `json:"foodcategory"`
	Quantity     int    `json:"quantity"`
}