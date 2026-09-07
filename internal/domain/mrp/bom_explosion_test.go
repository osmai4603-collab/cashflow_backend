package mrp

import "testing"

func TestConvertUoM(t *testing.T) {
	grams := UoM{ID: 1, Category: "weight", Ratio: 0.001, Rounding: 0.001}
	kg := UoM{ID: 2, Category: "weight", Ratio: 1, Rounding: 0.001}
	got, err := ConvertUoM(2500, grams, kg)
	if err != nil || got != 2.5 {
		t.Fatalf("converted quantity = %v (err: %v), want 2.5", got, err)
	}
	if _, err := ConvertUoM(1, grams, UoM{ID: 3, Category: "unit", Ratio: 1}); err == nil {
		t.Fatal("expected incompatible units to fail")
	}
}

func TestExplodeBoMFlattensPhantomAndKeepsNormal(t *testing.T) {
	root := &BillOfMaterials{ID: 1, ProductID: 100, ProductQty: 1, UoMID: 10, Type: BomTypeNormal}
	phantom := &BillOfMaterials{ID: 2, ProductID: 200, ProductQty: 2, UoMID: 10, Type: BomTypePhantom,
		Lines: []BomLine{{ProductID: 300, Quantity: 3, UoMID: 10}}}
	normal := &BillOfMaterials{ID: 3, ProductID: 400, ProductQty: 1, UoMID: 10, Type: BomTypeNormal,
		Lines: []BomLine{{ProductID: 500, Quantity: 4, UoMID: 10}}}
	boms := map[int64]*BillOfMaterials{200: phantom, 400: normal}
	units := map[int64]UoM{10: {ID: 10, Category: "unit", Ratio: 1, Rounding: 1}}
	root.Lines = []BomLine{{ProductID: 200, Quantity: 2, UoMID: 10}, {ProductID: 400, Quantity: 1, UoMID: 10}}

	components, err := ExplodeBoM(root, 2, func(productID int64) (*BillOfMaterials, error) {
		return boms[productID], nil
	}, func(id int64) (UoM, error) {
		return units[id], nil
	})
	if err != nil {
		t.Fatalf("explode BoM: %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("got %d components, want 2", len(components))
	}
	if components[0].ProductID != 300 || components[0].Quantity != 6 {
		t.Fatalf("phantom component = %+v, want product 300 quantity 6", components[0])
	}
	if components[1].ProductID != 400 || components[1].Quantity != 2 {
		t.Fatalf("normal subassembly = %+v, want product 400 quantity 2", components[1])
	}
}

func TestExplodeBoMRejectsCircularPhantom(t *testing.T) {
	a := &BillOfMaterials{ID: 1, ProductID: 100, ProductQty: 1, UoMID: 1, Type: BomTypePhantom}
	b := &BillOfMaterials{ID: 2, ProductID: 200, ProductQty: 1, UoMID: 1, Type: BomTypePhantom}
	a.Lines = []BomLine{{ProductID: 200, Quantity: 1, UoMID: 1}}
	b.Lines = []BomLine{{ProductID: 100, Quantity: 1, UoMID: 1}}
	_, err := ExplodeBoM(a, 1, func(productID int64) (*BillOfMaterials, error) {
		if productID == 200 {
			return b, nil
		}
		return a, nil
	}, func(id int64) (UoM, error) {
		return UoM{ID: id, Category: "unit", Ratio: 1, Rounding: 1}, nil
	})
	if err == nil {
		t.Fatal("expected circular BoM to fail")
	}
}
