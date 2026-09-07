package mrp

import (
	"fmt"
	"math"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// UoM describes the conversion data needed by BoM explosion.
type UoM struct {
	ID       int64
	Category string
	Ratio    float64
	Rounding float64
}

// ComponentResolver supplies a unit of measure by ID.
type ComponentResolver func(id int64) (UoM, error)

// BoMResolver supplies the applicable BoM for a product.
type BoMResolver func(productID int64) (*BillOfMaterials, error)

// ExplodedComponent is one executable raw-material requirement.
type ExplodedComponent struct {
	ProductID   int64
	UoMID       int64
	Quantity    float64
	OperationID *int64
	SourceBomID int64
	Level       int
}

// ConvertUoM converts a quantity between compatible units and applies target rounding.
func ConvertUoM(quantity float64, from, to UoM) (float64, error) {
	if from.ID <= 0 || to.ID <= 0 || from.Category == "" || to.Category == "" {
		return 0, platformerrors.Validation("invalid units of measure", nil)
	}
	if from.Category != to.Category {
		return 0, platformerrors.Validation("units of measure belong to different categories", nil)
	}
	if from.Ratio <= 0 || to.Ratio <= 0 {
		return 0, platformerrors.Validation("unit ratio must be positive", nil)
	}
	converted := quantity * from.Ratio / to.Ratio
	if to.Rounding > 0 {
		converted = math.Round(converted/to.Rounding) * to.Rounding
	}
	return converted, nil
}

// ExplodeBoM expands a root BoM into raw requirements for the requested quantity.
// Phantom BoMs are flattened; normal child BoMs remain executable subassembly requirements.
func ExplodeBoM(root *BillOfMaterials, targetQuantity float64, resolveBom BoMResolver, resolveUoM ComponentResolver) ([]ExplodedComponent, error) {
	if root == nil {
		return nil, platformerrors.Validation("root BoM is required", nil)
	}
	if err := root.Validate(); err != nil {
		return nil, err
	}
	if targetQuantity <= 0 {
		return nil, platformerrors.Validation("target quantity must be positive", nil)
	}
	if resolveBom == nil || resolveUoM == nil {
		return nil, platformerrors.Validation("BoM and UoM resolvers are required", nil)
	}
	if root.UoMID <= 0 {
		return nil, platformerrors.Validation("root BoM unit is required", nil)
	}

	factor := targetQuantity / root.ProductQty
	stack := map[int64]bool{root.ProductID: true}
	return explodeBoM(root, factor, 0, stack, resolveBom, resolveUoM)
}

func explodeBoM(bom *BillOfMaterials, factor float64, level int, stack map[int64]bool, resolveBom BoMResolver, resolveUoM ComponentResolver) ([]ExplodedComponent, error) {
	components := make([]ExplodedComponent, 0, len(bom.Lines))
	for _, line := range bom.Lines {
		if line.ProductID <= 0 || line.Quantity <= 0 || line.UoMID <= 0 {
			return nil, platformerrors.Validation("invalid BoM line", nil)
		}
		quantity := line.Quantity * factor
		child, err := resolveBom(line.ProductID)
		if err != nil {
			return nil, fmt.Errorf("resolve BoM for product %d: %w", line.ProductID, err)
		}
		if child == nil || child.Type != BomTypePhantom {
			components = append(components, ExplodedComponent{
				ProductID: line.ProductID, UoMID: line.UoMID, Quantity: quantity,
				OperationID: line.OperationID, SourceBomID: bom.ID, Level: level,
			})
			continue
		}
		if stack[line.ProductID] {
			return nil, platformerrors.Validation("circular BoM dependency", map[string]string{"product_id": fmt.Sprintf("%d", line.ProductID)})
		}
		from, err := resolveUoM(line.UoMID)
		if err != nil {
			return nil, fmt.Errorf("resolve BoM line unit %d: %w", line.UoMID, err)
		}
		to, err := resolveUoM(child.UoMID)
		if err != nil {
			return nil, fmt.Errorf("resolve child BoM unit %d: %w", child.UoMID, err)
		}
		childQuantity, err := ConvertUoM(quantity, from, to)
		if err != nil {
			return nil, fmt.Errorf("convert product %d quantity: %w", line.ProductID, err)
		}
		stack[line.ProductID] = true
		nested, err := explodeBoM(child, childQuantity/child.ProductQty, level+1, stack, resolveBom, resolveUoM)
		delete(stack, line.ProductID)
		if err != nil {
			return nil, err
		}
		components = append(components, nested...)
	}
	return components, nil
}
