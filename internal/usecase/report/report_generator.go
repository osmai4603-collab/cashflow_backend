package reportusecase

import (
	"context"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/report"
)

type generator struct {
	repo     report.Repository
	dataRepo report.DataRepository
}

func NewReportGenerator(repo report.Repository, dataRepo report.DataRepository) report.ReportGenerator {
	return &generator{repo: repo, dataRepo: dataRepo}
}

func (g *generator) Generate(ctx context.Context, r *report.Report, opts report.ReportOptions) (*report.ReportResult, error) {
	result := &report.ReportResult{
		ReportID: r.ID,
		Name:     r.Name,
		Options:  opts,
		Totals:   make(map[string]interface{}),
	}

	for _, col := range r.Columns {
		result.Columns = append(result.Columns, report.ReportColumnResult{
			Name:            col.Name,
			ExpressionLabel: col.ExpressionLabel,
			FigureType:      col.FigureType,
		})
	}

	// Collect prefixes
	prefixMap := make(map[string]bool)
	g.collectPrefixes(r.Lines, prefixMap)
	var prefixes []string
	for p := range prefixMap {
		prefixes = append(prefixes, p)
	}

	var balances map[string]float64
	var err error
	if len(prefixes) > 0 {
		balances, err = g.dataRepo.GetBalancesByAccountPrefix(ctx, prefixes, opts)
		if err != nil {
			return nil, err
		}
	} else {
		balances = make(map[string]float64)
	}

	// Compute lines
	lineMap := make(map[string]*report.ReportLineResult)
	result.Lines = g.computeLines(r.Lines, balances, lineMap)

	// Post-process aggregations
	g.resolveAggregations(result.Lines, lineMap, r.Lines)

	return result, nil
}

func (g *generator) collectPrefixes(lines []report.ReportLine, prefixMap map[string]bool) {
	for _, line := range lines {
		for _, expr := range line.Expressions {
			if expr.Engine == report.EngineAccountCodes {
				parts := strings.Split(expr.Formula, ",")
				for _, p := range parts {
					prefixMap[strings.TrimSpace(p)] = true
				}
			}
		}
		g.collectPrefixes(line.Children, prefixMap)
	}
}

func (g *generator) computeLines(lines []report.ReportLine, balances map[string]float64, lineMap map[string]*report.ReportLineResult) []report.ReportLineResult {
	var results []report.ReportLineResult
	for _, line := range lines {
		res := report.ReportLineResult{
			ID:         line.ID,
			Name:       line.Name,
			Code:       line.Code,
			Level:      line.HierarchyLevel,
			ParentID:   line.ParentID,
			Values:     make(map[string]interface{}),
			IsFoldable: line.Foldable,
		}

		for _, expr := range line.Expressions {
			if expr.Engine == report.EngineAccountCodes {
				var total float64
				parts := strings.Split(expr.Formula, ",")
				for _, p := range parts {
					total += balances[strings.TrimSpace(p)]
				}
				res.Values[expr.Label] = total
			}
		}

		if line.Code != "" {
			lineMap[line.Code] = &res
		}

		var childResults []report.ReportLineResult
		if len(line.Children) > 0 {
			res.HasChildren = true
			childResults = g.computeLines(line.Children, balances, lineMap)
		}

		results = append(results, res)
		results = append(results, childResults...)
	}
	return results
}

func (g *generator) resolveAggregations(results []report.ReportLineResult, lineMap map[string]*report.ReportLineResult, lines []report.ReportLine) {
	g.resolveLinesAggregations(lines, lineMap)
}

func (g *generator) resolveLinesAggregations(lines []report.ReportLine, lineMap map[string]*report.ReportLineResult) {
	for _, line := range lines {
		res, ok := lineMap[line.Code]
		if ok {
			for _, expr := range line.Expressions {
				if expr.Engine == report.EngineAggregation {
					res.Values[expr.Label] = g.evaluateAggregation(expr.Formula, lineMap)
				}
			}
		}
		g.resolveLinesAggregations(line.Children, lineMap)
	}
}

func (g *generator) evaluateAggregation(formula string, lineMap map[string]*report.ReportLineResult) float64 {
	parts := strings.Fields(formula)
	var total float64
	var op string = "+"

	for _, p := range parts {
		if p == "+" || p == "-" || p == "*" || p == "/" {
			op = p
			continue
		}

		var val float64
		if strings.Contains(p, ".") {
			dots := strings.Split(p, ".")
			code := dots[0]
			label := dots[1]
			if line, ok := lineMap[code]; ok {
				if v, ok := line.Values[label].(float64); ok {
					val = v
				}
			}
		} else {
			fmt.Sscanf(p, "%f", &val)
		}

		switch op {
		case "+":
			total += val
		case "-":
			total -= val
		case "*":
			total *= val
		case "/":
			if val != 0 {
				total /= val
			}
		}
	}

	return total
}
