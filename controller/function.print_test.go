package controller

import (
	"dinex-print-service/model"
	"strings"
	"testing"
)

func TestGenerateThermalKot(t *testing.T) {
	kds := model.Kds{
		ID:              "69cf45f156759eb2d9255b0b",
		OrderId:         "OID16252",
		RestaurantObjId: "699bd439f86553d0bc36b419",
		KdsNumber:       "KDS-878",
		TableNumber:     "Takeaway",
		KdsItems: []model.KdsItem{
			{
				ItemId:       "699c2b41f70fe3b2937c66df",
				Name:         "chiken biryani",
				FoodCategory: "NON-VEG",
				Quantity:     5,
			},
			{
				ItemId:       "699ea75b3ddfd1bd161a9517",
				Name:         "paneer chilly",
				FoodCategory: "VEG",
				Quantity:     3,
			},
		},
		Printed:   true,
		Status:    "PREPARING",
		CreatedAt: 1775191537,
		UpdatedAt: 1775192342,
	}

	// 1. Test 58mm KOT
	kot58 := GenerateThermalKot58mm(kds)
	t.Logf("Generated 58mm KOT:\n%s", kot58)
	if !strings.Contains(kot58, "KDS-878") {
		t.Error("58mm KOT does not contain KDS number")
	}
	if !strings.Contains(kot58, "order: OID16252-Takeaway") {
		t.Error("58mm KOT does not contain order info")
	}
	if !strings.Contains(kot58, "chiken biryani(NON-VEG)") && !strings.Contains(kot58, "chiken birya") {
		t.Error("58mm KOT does not contain first item name (or its trimmed variant)")
	}

	// 2. Test 80mm KOT
	kot80 := GenerateThermalKot80mm(kds)
	t.Logf("Generated 80mm KOT:\n%s", kot80)
	if !strings.Contains(kot80, "KDS-878") {
		t.Error("80mm KOT does not contain KDS number")
	}

	// 3. Test 112mm KOT
	kot112 := GenerateThermalKot112mm(kds)
	t.Logf("Generated 112mm KOT:\n%s", kot112)
	if !strings.Contains(kot112, "KDS-878") {
		t.Error("112mm KOT does not contain KDS number")
	}
}
