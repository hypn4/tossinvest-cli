package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockIndicators renders the investment-indicators payload as a sectioned
// table (one block per sectionName), full JSON, or CSV (section,key,value).
func WriteStockIndicators(w io.Writer, format Format, ind domain.StockIndicators) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ind)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "key", "value"}); err != nil {
			return err
		}
		names := sortedSectionNames(ind.Sections)
		for _, name := range names {
			keys := sortedMapKeys(ind.Sections[name])
			for _, k := range keys {
				if err := cw.Write([]string{name, k, fmt.Sprintf("%v", ind.Sections[name][k])}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — investment indicators\n", ind.ProductCode); err != nil {
			return err
		}
		order := []string{"가치평가", "수익", "배당", "안정성"}
		for _, name := range order {
			fields, ok := ind.Sections[name]
			if !ok {
				continue
			}
			if _, err := fmt.Fprintf(w, "\n=== %s ===\n", name); err != nil {
				return err
			}
			if err := renderIndicatorSection(w, name, fields); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func renderIndicatorSection(w io.Writer, name string, fields domain.IndicatorFields) error {
	type kv struct{ Label, Value string }
	var pairs []kv

	switch name {
	case "가치평가":
		pairs = []kv{
			{"PER", strVal(fields["displayPer"])},
			{"PBR", strVal(fields["displayPbr"])},
			{"PSR", strVal(fields["displayPsr"])},
		}
	case "수익":
		pairs = []kv{
			{"EPS", fmtUSDKRW(fields["eps"], fields["epsKrw"])},
			{"BPS", fmtUSDKRW(fields["bps"], fields["bpsKrw"])},
			{"ROE", strVal(fields["roe"])},
		}
	case "배당":
		pairs = []kv{
			{"배당 주기", strVal(fields["dividendFrequency"])},
			{"배당 수익률", fmtPctNum(fields["dividendYieldRatio"])},
			{"연간 배당금", fmtUSDKRWOrDash(fields["annualCash"], fields["annualCashKrw"])},
		}
	default:
		keys := sortedMapKeys(fields)
		for _, k := range keys {
			pairs = append(pairs, kv{k, strVal(fields[k])})
		}
	}

	headers := []string{"FIELD", "VALUE"}
	rows := make([][]string, len(pairs))
	for i, p := range pairs {
		rows[i] = []string{p.Label, p.Value}
	}
	return renderTable(w, headers, rows)
}

func strVal(v any) string {
	if v == nil {
		return "—"
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return fmt.Sprintf("%d", int64(s))
		}
		return fmt.Sprintf("%.2f", s)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func fmtUSDKRW(usdAny, krwAny any) string {
	usd, _ := usdAny.(float64)
	krw, _ := krwAny.(float64)
	return fmt.Sprintf("%s (₩%s)", formatUSD(usd), formatWithCommas(int64(krw)))
}

func fmtUSDKRWOrDash(usdAny, krwAny any) string {
	if usdAny == nil && krwAny == nil {
		return "—"
	}
	return fmtUSDKRW(usdAny, krwAny)
}

func fmtPctNum(v any) string {
	switch f := v.(type) {
	case float64:
		return fmt.Sprintf("%.2f%%", f)
	default:
		return strVal(v)
	}
}

func sortedSectionNames(m map[string]domain.IndicatorFields) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
