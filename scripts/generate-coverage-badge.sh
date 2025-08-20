#!/bin/bash

# Generate Coverage Badge Script
# This script generates a coverage badge SVG from coverage data

set -e

COVERAGE_FILE="coverage.out"
BADGE_FILE="coverage-badge.svg"
COVERAGE_PERCENTAGE=""

# Color thresholds
RED_THRESHOLD=50
YELLOW_THRESHOLD=70
GREEN_THRESHOLD=80

# Function to generate SVG badge
generate_badge() {
    local percentage="$1"
    local color="$2"
    local label="coverage"
    
    cat > "$BADGE_FILE" << EOF
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="104" height="20" role="img" aria-label="$label: $percentage%">
    <title>$label: $percentage%</title>
    <linearGradient id="s" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="r">
        <rect width="104" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#r)">
        <rect width="63" height="20" fill="#555"/>
        <rect x="63" width="41" height="20" fill="$color"/>
        <rect width="104" height="20" fill="url(#s)"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
        <text aria-hidden="true" x="325" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="530">$label</text>
        <text x="325" y="140" transform="scale(.1)" fill="#fff" textLength="530">$label</text>
        <text aria-hidden="true" x="825" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="310">$percentage%</text>
        <text x="825" y="140" transform="scale(.1)" fill="#fff" textLength="310">$percentage%</text>
    </g>
</svg>
EOF
}

# Function to determine color based on percentage
get_color() {
    local percentage="$1"
    
    if (( $(echo "$percentage >= $GREEN_THRESHOLD" | bc -l) )); then
        echo "#4c1"  # Green
    elif (( $(echo "$percentage >= $YELLOW_THRESHOLD" | bc -l) )); then
        echo "#dfb317"  # Yellow
    elif (( $(echo "$percentage >= $RED_THRESHOLD" | bc -l) )); then
        echo "#fe7d37"  # Orange
    else
        echo "#e05d44"  # Red
    fi
}

# Main function
main() {
    echo "Generating coverage badge..."
    
    # Check if coverage file exists
    if [[ ! -f "$COVERAGE_FILE" ]]; then
        echo "❌ Coverage file '$COVERAGE_FILE' not found"
        echo "Run tests with coverage first: go test -coverprofile=coverage.out ./..."
        exit 1
    fi
    
    # Extract coverage percentage
    COVERAGE_PERCENTAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')
    
    if [[ -z "$COVERAGE_PERCENTAGE" ]]; then
        echo "❌ Could not extract coverage percentage from $COVERAGE_FILE"
        exit 1
    fi
    
    echo "Coverage: $COVERAGE_PERCENTAGE%"
    
    # Determine badge color
    COLOR=$(get_color "$COVERAGE_PERCENTAGE")
    echo "Badge color: $COLOR"
    
    # Generate the badge
    generate_badge "$COVERAGE_PERCENTAGE" "$COLOR"
    
    echo "✅ Coverage badge generated: $BADGE_FILE"
    echo "Badge displays: $COVERAGE_PERCENTAGE% coverage"
    
    # Show file size
    if [[ -f "$BADGE_FILE" ]]; then
        SIZE=$(du -h "$BADGE_FILE" | cut -f1)
        echo "Badge file size: $SIZE"
    fi
}

# Show help
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat << EOF
Coverage Badge Generator

USAGE:
    $0 [OPTIONS]

DESCRIPTION:
    Generates an SVG coverage badge from Go coverage data.
    
    The badge color is determined by coverage percentage:
    - Red (< 50%): Poor coverage
    - Orange (50-69%): Fair coverage  
    - Yellow (70-79%): Good coverage
    - Green (≥ 80%): Excellent coverage

OPTIONS:
    -h, --help    Show this help message

EXAMPLES:
    # Generate coverage and badge
    go test -coverprofile=coverage.out ./...
    $0
    
    # Use in CI
    make test-coverage
    $0

OUTPUTS:
    - coverage-badge.svg: SVG badge file
    
REQUIREMENTS:
    - coverage.out file from 'go test -coverprofile'
    - bc calculator (for percentage comparisons)

EOF
    exit 0
fi

# Check for bc
if ! command -v bc >/dev/null 2>&1; then
    echo "❌ 'bc' calculator not found. Please install it:"
    echo "  macOS: brew install bc"
    echo "  Ubuntu: apt-get install bc"
    exit 1
fi

# Run main function
main "$@"