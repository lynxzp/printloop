# Average Print Position Design

## Goal

Add `.Positions.AveragePrintX` and `.Positions.AveragePrintY` to the template context so printer TOML templates can reference the average printing position across all print commands between EndInitSection and EndPrintSection.

## Definition

- **Qualifying print command**: G1 with positive E value AND at least one of X or Y present (same filter as FirstPrint/LastPrint).
- **Average**: Simple arithmetic mean. X and Y tracked independently (partial coords contribute only the axis they have).
- **Scope**: Commands between EndInitSection and EndPrintSection markers.

## Changes

1. **`MarkerPositions` struct** - Add `AveragePrintX float64` and `AveragePrintY float64` fields.
2. **`extractGCodeCoordinates`** - Add `sumX/countX/sumY/countY` accumulators in existing loop. Compute `sumX/countX` and `sumY/countY` at end. Return 2 extra float64 values.
3. **`findMarkerPositions` caller** - Receive extra return values, assign to `MarkerPositions` struct fields.
4. **Tests** - Add test cases for average calculation: multiple prints, partial coords, single print command.
5. **Templates** - No TOML changes needed; `.Positions.AveragePrintX` and `.Positions.AveragePrintY` become available automatically.

## Edge Cases

- **No print commands found**: Average defaults to 0.0 (same as FirstPrint/LastPrint behavior).
- **Single print command**: Average equals that command's X/Y.
- **Partial coordinates**: If a command has only X, it contributes to X average but not Y, and vice versa.
