package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const firstItemRow = 7

func fillTemplate(templatePath, outFile, name string, claimDate time.Time, expenses []Expense) error {
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return fmt.Errorf("open template: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	dateStyle, err := f.NewStyle(&excelize.Style{NumFmt: 15}) // d-mmm-yy
	if err != nil {
		return fmt.Errorf("create date style: %w", err)
	}
	moneyStyle, err := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
	if err != nil {
		return fmt.Errorf("create money style: %w", err)
	}

	if err := f.SetCellValue(sheet, "B2", name); err != nil {
		return err
	}
	if err := f.SetCellValue(sheet, "D2", claimDate); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "D2", "D2", dateStyle); err != nil {
		return err
	}

	for i, e := range expenses {
		row := firstItemRow + i
		aRef := fmt.Sprintf("A%d", row)
		bRef := fmt.Sprintf("B%d", row)
		dRef := fmt.Sprintf("D%d", row)

		if t, err := time.Parse("2006-01-02", e.Date); err == nil {
			if err := f.SetCellValue(sheet, aRef, t); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, aRef, aRef, dateStyle); err != nil {
				return err
			}
		} else {
			if err := f.SetCellValue(sheet, aRef, e.Date); err != nil {
				return err
			}
		}

		details := fmt.Sprintf("%s — %s", e.Vendor, e.Summary)
		if !strings.EqualFold(e.Currency, "GBP") {
			details = fmt.Sprintf("%s [%.2f %s]", details, e.Amount, strings.ToUpper(e.Currency))
		}
		if err := f.SetCellValue(sheet, bRef, details); err != nil {
			return err
		}

		if strings.EqualFold(e.Currency, "GBP") {
			if err := f.SetCellValue(sheet, dRef, e.Amount); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, dRef, dRef, moneyStyle); err != nil {
				return err
			}
		}
	}

	return f.SaveAs(outFile)
}
