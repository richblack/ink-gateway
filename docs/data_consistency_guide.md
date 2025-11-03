# Data Consistency Checking and Repair Process Documentation

## Table of Contents

1. [Overview](#overview)
2. [Data Consistency Framework](#data-consistency-framework)
3. [Automated Consistency Checks](#automated-consistency-checks)
4. [Manual Consistency Verification](#manual-consistency-verification)
5. [Data Repair Procedures](#data-repair-procedures)
6. [Preventive Measures](#preventive-measures)
7. [Monitoring and Alerting](#monitoring-and-alerting)
8. [Recovery Procedures](#recovery-procedures)
9. [Consistency Check Automation](#consistency-check-automation)
10. [Best Practices](#best-practices)

## Overview

This guide provides comprehensive procedures for maintaining data consistency in the Semantic Text Processor system, particularly focusing on the unified chunk system and multi-database architecture. Data consistency is critical for system reliability and user trust.

### Data Consistency Scope

The system maintains consistency across:
- **Hierarchical Relationships**: Parent-child chunk relationships
- **Template References**: Template-instance relationships
- **Tag Associations**: Chunk-tag mappings
- **Embedding Synchronization**: Chunk-embedding relationships
- **Graph Relationships**: Knowledge graph node-edge consistency
- **Cross-table References**: Foreign key integrity

### Consistency Levels

1. **Strong Consistency**: Critical relationships that must always be valid
2. **Eventual Consistency**: Derived data that can be rebuilt
3. **Soft Consistency**: Performance optimizations that can tolerate temporary inconsistencies

## Data Consistency Framework

### Core Consistency Rules

**1. Hierarchical Integrity Rules**:
```sql
-- Rule 1: No circular references in chunk hierarchy
WITH RECURSIVE chunk_hierarchy AS (
    SELECT id, parent_chunk_id, 1 as depth, ARRAY[id] as path
    FROM chunks
    WHERE parent_chunk_id IS NULL

    UNION ALL

    SELECT c.id, c.parent_chunk_id, ch.depth + 1, ch.path || c.id
    FROM chunks c
    JOIN chunk_hierarchy ch ON c.parent_chunk_id = ch.id
    WHERE c.id != ALL(ch.path) AND ch.depth < 50
)
SELECT * FROM chunk_hierarchy WHERE depth > 20; -- Flag potential circular refs

-- Rule 2: Parent chunks must exist
SELECT COUNT(*) as orphaned_chunks
FROM chunks c
WHERE c.parent_chunk_id IS NOT NULL
AND c.parent_chunk_id NOT IN (SELECT id FROM chunks);

-- Rule 3: Indent level must be consistent with hierarchy depth
WITH RECURSIVE hierarchy_depth AS (
    SELECT id, parent_chunk_id, indent_level, 0 as calculated_depth
    FROM chunks
    WHERE parent_chunk_id IS NULL

    UNION ALL

    SELECT c.id, c.parent_chunk_id, c.indent_level, hd.calculated_depth + 1
    FROM chunks c
    JOIN hierarchy_depth hd ON c.parent_chunk_id = hd.id
)
SELECT COUNT(*) as inconsistent_indent_levels
FROM hierarchy_depth
WHERE indent_level != calculated_depth;
```

**2. Template Consistency Rules**:
```sql
-- Rule 1: Template chunks must have is_template = true
SELECT COUNT(*) as invalid_templates
FROM chunks
WHERE template_chunk_id IS NOT NULL
AND template_chunk_id NOT IN (
    SELECT id FROM chunks WHERE is_template = true
);

-- Rule 2: Slot chunks must have valid parent template
SELECT COUNT(*) as orphaned_slots
FROM chunks
WHERE is_slot = true
AND (parent_chunk_id IS NULL OR
     parent_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true));

-- Rule 3: Template instances must reference valid templates
SELECT COUNT(*) as invalid_template_instances
FROM chunks
WHERE template_chunk_id IS NOT NULL
AND template_chunk_id NOT IN (SELECT id FROM chunks);
```

**3. Embedding Consistency Rules**:
```sql
-- Rule 1: Every non-template chunk should have embeddings
SELECT COUNT(*) as missing_embeddings
FROM chunks c
LEFT JOIN embeddings e ON c.id = e.chunk_id
WHERE c.is_template = false
AND c.is_slot = false
AND e.chunk_id IS NULL;

-- Rule 2: Embeddings must reference valid chunks
SELECT COUNT(*) as orphaned_embeddings
FROM embeddings e
WHERE e.chunk_id NOT IN (SELECT id FROM chunks);

-- Rule 3: Vector dimensions must be consistent
SELECT model_name, vector_length, COUNT(*) as count
FROM (
    SELECT model_name, array_length(vector, 1) as vector_length
    FROM embeddings
) stats
GROUP BY model_name, vector_length
HAVING COUNT(*) > 1; -- Inconsistent dimensions per model
```

### Consistency Check Categories

**1. Critical Consistency Checks** (Must be resolved immediately):
- Orphaned chunks with invalid parent references
- Circular references in hierarchy
- Invalid template references
- Missing required embeddings for searchable content

**2. Warning-Level Checks** (Should be resolved):
- Inconsistent indent levels
- Orphaned embeddings
- Unused templates
- Stale cache entries

**3. Informational Checks** (Monitor but may not require action):
- Performance-related inconsistencies
- Optimization opportunities
- Data distribution anomalies

## Automated Consistency Checks

### Daily Consistency Check Script

```bash
#!/bin/bash
# daily_consistency_check.sh

set -e

LOG_FILE="/var/log/consistency_check_$(date +%Y%m%d).log"
REPORT_FILE="/var/log/consistency_report_$(date +%Y%m%d).json"
API_BASE="http://localhost:8080/api/v1"

echo "Starting daily consistency check at $(date)" | tee "$LOG_FILE"

# Initialize report structure
cat > "$REPORT_FILE" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "checks": {},
    "summary": {
        "total_checks": 0,
        "passed": 0,
        "warnings": 0,
        "failures": 0
    },
    "recommendations": []
}
EOF

# Function to run SQL consistency check
run_sql_check() {
    local check_name="$1"
    local query="$2"
    local threshold="$3"
    local severity="$4"

    echo "Running check: $check_name" | tee -a "$LOG_FILE"

    # Execute query via API (assuming admin endpoint exists)
    result=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"$query\"}" | jq -r '.result[0].count // 0')

    # Determine status
    if [ "$result" -eq 0 ]; then
        status="PASS"
    elif [ "$result" -le "$threshold" ]; then
        status="WARNING"
    else
        status="FAIL"
    fi

    echo "  Result: $result (threshold: $threshold, status: $status)" | tee -a "$LOG_FILE"

    # Update report
    jq --arg check "$check_name" \
       --arg status "$status" \
       --arg result "$result" \
       --arg threshold "$threshold" \
       --arg severity "$severity" \
       '.checks[$check] = {
           "status": $status,
           "result": ($result | tonumber),
           "threshold": ($threshold | tonumber),
           "severity": $severity,
           "timestamp": now | strftime("%Y-%m-%dT%H:%M:%SZ")
       }' "$REPORT_FILE" > "${REPORT_FILE}.tmp" && mv "${REPORT_FILE}.tmp" "$REPORT_FILE"

    return $([ "$status" = "FAIL" ] && echo 1 || echo 0)
}

# Run consistency checks
TOTAL_CHECKS=0
FAILED_CHECKS=0

# Check 1: Orphaned chunks
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if run_sql_check "orphaned_chunks" \
    "SELECT COUNT(*) as count FROM chunks WHERE parent_chunk_id IS NOT NULL AND parent_chunk_id NOT IN (SELECT id FROM chunks)" \
    "0" "critical"; then
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check 2: Circular references
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if run_sql_check "circular_references" \
    "WITH RECURSIVE hierarchy AS (SELECT id, parent_chunk_id, 1 as depth FROM chunks WHERE parent_chunk_id IS NULL UNION ALL SELECT c.id, c.parent_chunk_id, h.depth + 1 FROM chunks c JOIN hierarchy h ON c.parent_chunk_id = h.id WHERE h.depth < 20) SELECT COUNT(*) as count FROM hierarchy WHERE depth >= 15" \
    "0" "critical"; then
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check 3: Invalid template references
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if run_sql_check "invalid_template_refs" \
    "SELECT COUNT(*) as count FROM chunks WHERE template_chunk_id IS NOT NULL AND template_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true)" \
    "0" "critical"; then
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check 4: Missing embeddings
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if run_sql_check "missing_embeddings" \
    "SELECT COUNT(*) as count FROM chunks c LEFT JOIN embeddings e ON c.id = e.chunk_id WHERE c.is_template = false AND c.is_slot = false AND e.chunk_id IS NULL" \
    "100" "warning"; then
    # Non-critical, just log
    echo "Warning: Missing embeddings detected" | tee -a "$LOG_FILE"
fi

# Check 5: Orphaned embeddings
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
if run_sql_check "orphaned_embeddings" \
    "SELECT COUNT(*) as count FROM embeddings WHERE chunk_id NOT IN (SELECT id FROM chunks)" \
    "0" "warning"; then
    echo "Warning: Orphaned embeddings detected" | tee -a "$LOG_FILE"
fi

# Generate summary
PASSED_CHECKS=$((TOTAL_CHECKS - FAILED_CHECKS))
jq --arg total "$TOTAL_CHECKS" \
   --arg passed "$PASSED_CHECKS" \
   --arg failed "$FAILED_CHECKS" \
   '.summary.total_checks = ($total | tonumber) |
    .summary.passed = ($passed | tonumber) |
    .summary.failures = ($failed | tonumber)' \
    "$REPORT_FILE" > "${REPORT_FILE}.tmp" && mv "${REPORT_FILE}.tmp" "$REPORT_FILE"

echo "" | tee -a "$LOG_FILE"
echo "Consistency check completed at $(date)" | tee -a "$LOG_FILE"
echo "Total checks: $TOTAL_CHECKS, Passed: $PASSED_CHECKS, Failed: $FAILED_CHECKS" | tee -a "$LOG_FILE"

# Send alert if critical failures detected
if [ "$FAILED_CHECKS" -gt 0 ]; then
    echo "ALERT: $FAILED_CHECKS critical consistency check(s) failed!" | tee -a "$LOG_FILE"

    # Send notification (customize for your notification system)
    curl -X POST "http://your-notification-service/alert" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Data Consistency Check Failed\",
            \"message\": \"$FAILED_CHECKS critical consistency issues detected\",
            \"severity\": \"high\",
            \"report_file\": \"$REPORT_FILE\"
        }" || echo "Failed to send alert notification"

    exit 1
else
    echo "All critical consistency checks passed!" | tee -a "$LOG_FILE"
    exit 0
fi
```

### Real-time Consistency Monitoring

```python
# consistency_monitor.py
import psycopg2
import json
import time
import logging
from datetime import datetime
import requests

class ConsistencyMonitor:
    def __init__(self, db_config, alert_webhook=None):
        self.db_config = db_config
        self.alert_webhook = alert_webhook
        self.logger = self._setup_logging()
        self.checks = {
            'orphaned_chunks': {
                'query': """
                    SELECT COUNT(*) as count FROM chunks
                    WHERE parent_chunk_id IS NOT NULL
                    AND parent_chunk_id NOT IN (SELECT id FROM chunks)
                """,
                'threshold': 0,
                'severity': 'critical'
            },
            'invalid_templates': {
                'query': """
                    SELECT COUNT(*) as count FROM chunks
                    WHERE template_chunk_id IS NOT NULL
                    AND template_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true)
                """,
                'threshold': 0,
                'severity': 'critical'
            },
            'missing_embeddings': {
                'query': """
                    SELECT COUNT(*) as count FROM chunks c
                    LEFT JOIN embeddings e ON c.id = e.chunk_id
                    WHERE c.is_template = false AND c.is_slot = false
                    AND e.chunk_id IS NULL
                """,
                'threshold': 50,
                'severity': 'warning'
            }
        }

    def _setup_logging(self):
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler('/var/log/consistency_monitor.log'),
                logging.StreamHandler()
            ]
        )
        return logging.getLogger(__name__)

    def connect_db(self):
        """Connect to database"""
        try:
            return psycopg2.connect(**self.db_config)
        except Exception as e:
            self.logger.error(f"Database connection failed: {e}")
            return None

    def run_check(self, check_name, check_config):
        """Run a single consistency check"""
        conn = self.connect_db()
        if not conn:
            return None

        try:
            with conn.cursor() as cursor:
                cursor.execute(check_config['query'])
                result = cursor.fetchone()[0]

                status = 'pass'
                if result > check_config['threshold']:
                    status = 'fail' if check_config['severity'] == 'critical' else 'warning'

                return {
                    'check_name': check_name,
                    'result': result,
                    'threshold': check_config['threshold'],
                    'status': status,
                    'severity': check_config['severity'],
                    'timestamp': datetime.now().isoformat()
                }
        except Exception as e:
            self.logger.error(f"Check {check_name} failed: {e}")
            return None
        finally:
            conn.close()

    def run_all_checks(self):
        """Run all consistency checks"""
        results = {}

        for check_name, check_config in self.checks.items():
            self.logger.info(f"Running check: {check_name}")
            result = self.run_check(check_name, check_config)

            if result:
                results[check_name] = result

                if result['status'] == 'fail':
                    self.logger.error(f"CRITICAL: {check_name} failed with {result['result']} issues")
                    self.send_alert(result)
                elif result['status'] == 'warning':
                    self.logger.warning(f"WARNING: {check_name} has {result['result']} issues")

        return results

    def send_alert(self, check_result):
        """Send alert for failed checks"""
        if not self.alert_webhook:
            return

        alert_data = {
            'title': f"Data Consistency Alert: {check_result['check_name']}",
            'message': f"Check failed with {check_result['result']} issues (threshold: {check_result['threshold']})",
            'severity': check_result['severity'],
            'timestamp': check_result['timestamp'],
            'check_details': check_result
        }

        try:
            response = requests.post(self.alert_webhook, json=alert_data, timeout=10)
            response.raise_for_status()
            self.logger.info(f"Alert sent for {check_result['check_name']}")
        except Exception as e:
            self.logger.error(f"Failed to send alert: {e}")

    def continuous_monitoring(self, interval_minutes=15):
        """Run continuous monitoring with specified interval"""
        self.logger.info(f"Starting continuous monitoring with {interval_minutes}-minute intervals")

        while True:
            try:
                results = self.run_all_checks()

                # Log summary
                total_checks = len(results)
                failed_checks = sum(1 for r in results.values() if r['status'] == 'fail')
                warning_checks = sum(1 for r in results.values() if r['status'] == 'warning')

                self.logger.info(f"Monitoring cycle complete: {total_checks} checks, {failed_checks} failures, {warning_checks} warnings")

                # Save results
                with open(f'/var/log/consistency_results_{datetime.now().strftime("%Y%m%d_%H%M")}.json', 'w') as f:
                    json.dump(results, f, indent=2)

                time.sleep(interval_minutes * 60)

            except KeyboardInterrupt:
                self.logger.info("Monitoring stopped by user")
                break
            except Exception as e:
                self.logger.error(f"Monitoring error: {e}")
                time.sleep(60)  # Wait 1 minute before retrying

# Usage example
if __name__ == "__main__":
    db_config = {
        'host': 'localhost',
        'database': 'semantic_text_processor',
        'user': 'your_user',
        'password': 'your_password'
    }

    monitor = ConsistencyMonitor(
        db_config=db_config,
        alert_webhook='http://your-notification-service/webhook'
    )

    # Run once
    results = monitor.run_all_checks()
    print(json.dumps(results, indent=2))

    # Or run continuously
    # monitor.continuous_monitoring(interval_minutes=15)
```

## Manual Consistency Verification

### Interactive Consistency Check Tool

```bash
#!/bin/bash
# interactive_consistency_check.sh

set -e

API_BASE="http://localhost:8080/api/v1"
TEMP_DIR="/tmp/consistency_check_$$"
mkdir -p "$TEMP_DIR"

echo "=== Semantic Text Processor - Interactive Consistency Check ==="
echo ""

# Function to display menu
show_menu() {
    echo "Select consistency check to run:"
    echo "1. Hierarchical integrity check"
    echo "2. Template consistency check"
    echo "3. Embedding synchronization check"
    echo "4. Graph consistency check"
    echo "5. Full comprehensive check"
    echo "6. Generate detailed report"
    echo "7. Exit"
    echo ""
    read -p "Enter your choice (1-7): " choice
    echo ""
}

# Function to run hierarchical integrity check
check_hierarchical_integrity() {
    echo "Running hierarchical integrity check..."

    # Check for orphaned chunks
    orphaned=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM chunks WHERE parent_chunk_id IS NOT NULL AND parent_chunk_id NOT IN (SELECT id FROM chunks)"}' | \
        jq -r '.result[0].count // 0')

    echo "Orphaned chunks: $orphaned"

    # Check for circular references
    echo "Checking for circular references..."
    deep_hierarchies=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "WITH RECURSIVE hierarchy AS (SELECT id, parent_chunk_id, 1 as depth FROM chunks WHERE parent_chunk_id IS NULL UNION ALL SELECT c.id, c.parent_chunk_id, h.depth + 1 FROM chunks c JOIN hierarchy h ON c.parent_chunk_id = h.id WHERE h.depth < 20) SELECT COUNT(*) as count FROM hierarchy WHERE depth >= 15"}' | \
        jq -r '.result[0].count // 0')

    echo "Potentially problematic deep hierarchies: $deep_hierarchies"

    # Check indent level consistency
    echo "Checking indent level consistency..."
    # This would require a more complex query to calculate expected vs actual indent levels

    if [ "$orphaned" -eq 0 ] && [ "$deep_hierarchies" -eq 0 ]; then
        echo "✅ Hierarchical integrity check PASSED"
    else
        echo "❌ Hierarchical integrity check FAILED"
        echo "  - Orphaned chunks: $orphaned"
        echo "  - Deep hierarchies: $deep_hierarchies"
    fi
    echo ""
}

# Function to check template consistency
check_template_consistency() {
    echo "Running template consistency check..."

    # Check invalid template references
    invalid_templates=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM chunks WHERE template_chunk_id IS NOT NULL AND template_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true)"}' | \
        jq -r '.result[0].count // 0')

    echo "Invalid template references: $invalid_templates"

    # Check orphaned slots
    orphaned_slots=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM chunks WHERE is_slot = true AND (parent_chunk_id IS NULL OR parent_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true))"}' | \
        jq -r '.result[0].count // 0')

    echo "Orphaned template slots: $orphaned_slots"

    # Get template usage statistics
    echo "Template usage statistics:"
    curl -s "$API_BASE/templates" | jq -r '.[] | "Template: \(.template_name), Instances: \(.instance_count // 0)"'

    if [ "$invalid_templates" -eq 0 ] && [ "$orphaned_slots" -eq 0 ]; then
        echo "✅ Template consistency check PASSED"
    else
        echo "❌ Template consistency check FAILED"
        echo "  - Invalid template references: $invalid_templates"
        echo "  - Orphaned slots: $orphaned_slots"
    fi
    echo ""
}

# Function to check embedding synchronization
check_embedding_sync() {
    echo "Running embedding synchronization check..."

    # Check for missing embeddings
    missing_embeddings=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM chunks c LEFT JOIN embeddings e ON c.id = e.chunk_id WHERE c.is_template = false AND c.is_slot = false AND e.chunk_id IS NULL"}' | \
        jq -r '.result[0].count // 0')

    echo "Chunks missing embeddings: $missing_embeddings"

    # Check for orphaned embeddings
    orphaned_embeddings=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM embeddings WHERE chunk_id NOT IN (SELECT id FROM chunks)"}' | \
        jq -r '.result[0].count // 0')

    echo "Orphaned embeddings: $orphaned_embeddings"

    # Check embedding model consistency
    echo "Embedding model distribution:"
    curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT model_name, COUNT(*) as count FROM embeddings GROUP BY model_name ORDER BY count DESC"}' | \
        jq -r '.result[] | "  \(.model_name): \(.count) embeddings"'

    if [ "$missing_embeddings" -eq 0 ] && [ "$orphaned_embeddings" -eq 0 ]; then
        echo "✅ Embedding synchronization check PASSED"
    else
        echo "⚠️ Embedding synchronization issues detected"
        echo "  - Missing embeddings: $missing_embeddings"
        echo "  - Orphaned embeddings: $orphaned_embeddings"
    fi
    echo ""
}

# Function to check graph consistency
check_graph_consistency() {
    echo "Running graph consistency check..."

    # Check for orphaned graph nodes
    orphaned_nodes=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM graph_nodes WHERE chunk_id NOT IN (SELECT id FROM chunks)"}' | \
        jq -r '.result[0].count // 0')

    echo "Orphaned graph nodes: $orphaned_nodes"

    # Check for edges with missing nodes
    invalid_edges=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM graph_edges WHERE source_node_id NOT IN (SELECT id FROM graph_nodes) OR target_node_id NOT IN (SELECT id FROM graph_nodes)"}' | \
        jq -r '.result[0].count // 0')

    echo "Invalid graph edges: $invalid_edges"

    # Get graph statistics
    echo "Graph statistics:"
    total_nodes=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM graph_nodes"}' | \
        jq -r '.result[0].count // 0')

    total_edges=$(curl -s -X POST "$API_BASE/admin/query" \
        -H "Content-Type: application/json" \
        -d '{"query": "SELECT COUNT(*) as count FROM graph_edges"}' | \
        jq -r '.result[0].count // 0')

    echo "  Total nodes: $total_nodes"
    echo "  Total edges: $total_edges"
    echo "  Average connections per node: $(echo "scale=2; $total_edges * 2 / $total_nodes" | bc -l 2>/dev/null || echo "N/A")"

    if [ "$orphaned_nodes" -eq 0 ] && [ "$invalid_edges" -eq 0 ]; then
        echo "✅ Graph consistency check PASSED"
    else
        echo "❌ Graph consistency check FAILED"
        echo "  - Orphaned nodes: $orphaned_nodes"
        echo "  - Invalid edges: $invalid_edges"
    fi
    echo ""
}

# Function to run full comprehensive check
run_full_check() {
    echo "Running comprehensive consistency check..."
    echo "=" | tr '=' '-' | head -c 50; echo ""

    check_hierarchical_integrity
    check_template_consistency
    check_embedding_sync
    check_graph_consistency

    echo "Comprehensive check completed!"
    echo ""
}

# Function to generate detailed report
generate_report() {
    echo "Generating detailed consistency report..."

    REPORT_FILE="$TEMP_DIR/consistency_report_$(date +%Y%m%d_%H%M%S).json"

    # Collect all consistency data
    cat > "$REPORT_FILE" << EOF
{
    "report_timestamp": "$(date -Iseconds)",
    "system_info": {
        "version": "$(curl -s $API_BASE/health | jq -r '.version // "unknown"')",
        "uptime": "$(curl -s $API_BASE/health | jq -r '.uptime // "unknown"')"
    },
    "consistency_checks": {},
    "recommendations": []
}
EOF

    echo "Report generated: $REPORT_FILE"
    echo "Report content:"
    cat "$REPORT_FILE" | jq '.'
    echo ""
}

# Main loop
while true; do
    show_menu

    case $choice in
        1)
            check_hierarchical_integrity
            ;;
        2)
            check_template_consistency
            ;;
        3)
            check_embedding_sync
            ;;
        4)
            check_graph_consistency
            ;;
        5)
            run_full_check
            ;;
        6)
            generate_report
            ;;
        7)
            echo "Exiting..."
            rm -rf "$TEMP_DIR"
            exit 0
            ;;
        *)
            echo "Invalid choice. Please try again."
            echo ""
            ;;
    esac

    read -p "Press Enter to continue..."
    echo ""
done
```

## Data Repair Procedures

### Automated Repair Scripts

```python
# data_repair.py
import psycopg2
import logging
import json
from datetime import datetime
import requests

class DataRepairManager:
    def __init__(self, db_config, dry_run=True):
        self.db_config = db_config
        self.dry_run = dry_run
        self.logger = self._setup_logging()
        self.repair_log = []

    def _setup_logging(self):
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler('/var/log/data_repair.log'),
                logging.StreamHandler()
            ]
        )
        return logging.getLogger(__name__)

    def connect_db(self):
        try:
            return psycopg2.connect(**self.db_config)
        except Exception as e:
            self.logger.error(f"Database connection failed: {e}")
            return None

    def log_repair_action(self, action_type, description, affected_rows=0, sql_query=None):
        """Log repair action for audit trail"""
        log_entry = {
            'timestamp': datetime.now().isoformat(),
            'action_type': action_type,
            'description': description,
            'affected_rows': affected_rows,
            'sql_query': sql_query,
            'dry_run': self.dry_run
        }
        self.repair_log.append(log_entry)
        self.logger.info(f"{'[DRY RUN] ' if self.dry_run else ''}{action_type}: {description} (affected: {affected_rows})")

    def repair_orphaned_chunks(self):
        """Repair chunks with invalid parent references"""
        conn = self.connect_db()
        if not conn:
            return False

        try:
            with conn.cursor() as cursor:
                # Find orphaned chunks
                cursor.execute("""
                    SELECT id, parent_chunk_id, content
                    FROM chunks
                    WHERE parent_chunk_id IS NOT NULL
                    AND parent_chunk_id NOT IN (SELECT id FROM chunks)
                """)
                orphaned_chunks = cursor.fetchall()

                if not orphaned_chunks:
                    self.log_repair_action("CHECK", "No orphaned chunks found")
                    return True

                self.logger.warning(f"Found {len(orphaned_chunks)} orphaned chunks")

                for chunk_id, invalid_parent_id, content in orphaned_chunks:
                    # Strategy 1: Try to find a similar parent by content or hierarchy
                    cursor.execute("""
                        SELECT id FROM chunks
                        WHERE text_id = (SELECT text_id FROM chunks WHERE id = %s)
                        AND parent_chunk_id IS NULL
                        ORDER BY created_at
                        LIMIT 1
                    """, (chunk_id,))

                    potential_parent = cursor.fetchone()

                    if potential_parent:
                        # Set parent to root of same text
                        repair_sql = """
                            UPDATE chunks
                            SET parent_chunk_id = %s,
                                indent_level = GREATEST(
                                    (SELECT COALESCE(indent_level, 0) FROM chunks WHERE id = %s) + 1,
                                    1
                                )
                            WHERE id = %s
                        """

                        if not self.dry_run:
                            cursor.execute(repair_sql, (potential_parent[0], potential_parent[0], chunk_id))

                        self.log_repair_action(
                            "REPAIR",
                            f"Orphaned chunk {chunk_id} reparented to {potential_parent[0]}",
                            1,
                            repair_sql
                        )
                    else:
                        # Strategy 2: Set as root chunk
                        repair_sql = """
                            UPDATE chunks
                            SET parent_chunk_id = NULL, indent_level = 0
                            WHERE id = %s
                        """

                        if not self.dry_run:
                            cursor.execute(repair_sql, (chunk_id,))

                        self.log_repair_action(
                            "REPAIR",
                            f"Orphaned chunk {chunk_id} converted to root chunk",
                            1,
                            repair_sql
                        )

                if not self.dry_run:
                    conn.commit()
                    self.logger.info(f"Repaired {len(orphaned_chunks)} orphaned chunks")
                else:
                    self.logger.info(f"[DRY RUN] Would repair {len(orphaned_chunks)} orphaned chunks")

                return True

        except Exception as e:
            self.logger.error(f"Error repairing orphaned chunks: {e}")
            if conn:
                conn.rollback()
            return False
        finally:
            if conn:
                conn.close()

    def repair_invalid_template_references(self):
        """Repair chunks with invalid template references"""
        conn = self.connect_db()
        if not conn:
            return False

        try:
            with conn.cursor() as cursor:
                # Find chunks with invalid template references
                cursor.execute("""
                    SELECT id, template_chunk_id
                    FROM chunks
                    WHERE template_chunk_id IS NOT NULL
                    AND template_chunk_id NOT IN (SELECT id FROM chunks WHERE is_template = true)
                """)
                invalid_refs = cursor.fetchall()

                if not invalid_refs:
                    self.log_repair_action("CHECK", "No invalid template references found")
                    return True

                for chunk_id, invalid_template_id in invalid_refs:
                    # Strategy: Remove invalid template reference
                    repair_sql = """
                        UPDATE chunks
                        SET template_chunk_id = NULL, slot_value = NULL
                        WHERE id = %s
                    """

                    if not self.dry_run:
                        cursor.execute(repair_sql, (chunk_id,))

                    self.log_repair_action(
                        "REPAIR",
                        f"Removed invalid template reference from chunk {chunk_id}",
                        1,
                        repair_sql
                    )

                if not self.dry_run:
                    conn.commit()
                    self.logger.info(f"Repaired {len(invalid_refs)} invalid template references")

                return True

        except Exception as e:
            self.logger.error(f"Error repairing template references: {e}")
            if conn:
                conn.rollback()
            return False
        finally:
            if conn:
                conn.close()

    def repair_orphaned_embeddings(self):
        """Remove embeddings for non-existent chunks"""
        conn = self.connect_db()
        if not conn:
            return False

        try:
            with conn.cursor() as cursor:
                # Find orphaned embeddings
                cursor.execute("""
                    SELECT id, chunk_id
                    FROM embeddings
                    WHERE chunk_id NOT IN (SELECT id FROM chunks)
                """)
                orphaned_embeddings = cursor.fetchall()

                if not orphaned_embeddings:
                    self.log_repair_action("CHECK", "No orphaned embeddings found")
                    return True

                # Remove orphaned embeddings
                repair_sql = """
                    DELETE FROM embeddings
                    WHERE chunk_id NOT IN (SELECT id FROM chunks)
                """

                if not self.dry_run:
                    cursor.execute(repair_sql)
                    affected_rows = cursor.rowcount
                    conn.commit()
                    self.logger.info(f"Removed {affected_rows} orphaned embeddings")
                else:
                    affected_rows = len(orphaned_embeddings)
                    self.logger.info(f"[DRY RUN] Would remove {affected_rows} orphaned embeddings")

                self.log_repair_action(
                    "REPAIR",
                    "Removed orphaned embeddings",
                    affected_rows,
                    repair_sql
                )

                return True

        except Exception as e:
            self.logger.error(f"Error repairing orphaned embeddings: {e}")
            if conn:
                conn.rollback()
            return False
        finally:
            if conn:
                conn.close()

    def repair_missing_embeddings(self, limit=100):
        """Generate embeddings for chunks that are missing them"""
        conn = self.connect_db()
        if not conn:
            return False

        try:
            with conn.cursor() as cursor:
                # Find chunks missing embeddings
                cursor.execute("""
                    SELECT c.id, c.content
                    FROM chunks c
                    LEFT JOIN embeddings e ON c.id = e.chunk_id
                    WHERE c.is_template = false
                    AND c.is_slot = false
                    AND e.chunk_id IS NULL
                    AND LENGTH(c.content) > 10
                    LIMIT %s
                """, (limit,))

                missing_embeddings = cursor.fetchall()

                if not missing_embeddings:
                    self.log_repair_action("CHECK", "No missing embeddings found")
                    return True

                self.logger.info(f"Found {len(missing_embeddings)} chunks missing embeddings")

                # Generate embeddings (this would call your embedding service)
                for chunk_id, content in missing_embeddings:
                    if not self.dry_run:
                        # Call embedding API
                        embedding_response = self._generate_embedding(content)
                        if embedding_response:
                            cursor.execute("""
                                INSERT INTO embeddings (chunk_id, vector, model_name, created_at)
                                VALUES (%s, %s, %s, NOW())
                            """, (chunk_id, embedding_response['vector'], embedding_response['model']))

                    self.log_repair_action(
                        "REPAIR",
                        f"Generated embedding for chunk {chunk_id}",
                        1
                    )

                if not self.dry_run:
                    conn.commit()
                    self.logger.info(f"Generated {len(missing_embeddings)} missing embeddings")

                return True

        except Exception as e:
            self.logger.error(f"Error generating missing embeddings: {e}")
            if conn:
                conn.rollback()
            return False
        finally:
            if conn:
                conn.close()

    def _generate_embedding(self, content):
        """Generate embedding for content (placeholder - implement with your embedding service)"""
        # This would call your actual embedding service
        # For now, return a placeholder
        return {
            'vector': [0.0] * 1536,  # Placeholder vector
            'model': 'text-embedding-ada-002'
        }

    def run_all_repairs(self):
        """Run all repair procedures"""
        self.logger.info(f"Starting data repair process ({'DRY RUN' if self.dry_run else 'LIVE RUN'})")

        repair_functions = [
            ('Orphaned Chunks', self.repair_orphaned_chunks),
            ('Invalid Template References', self.repair_invalid_template_references),
            ('Orphaned Embeddings', self.repair_orphaned_embeddings),
            ('Missing Embeddings', lambda: self.repair_missing_embeddings(limit=50))
        ]

        results = {}
        for repair_name, repair_func in repair_functions:
            self.logger.info(f"Running repair: {repair_name}")
            try:
                success = repair_func()
                results[repair_name] = 'SUCCESS' if success else 'FAILED'
            except Exception as e:
                self.logger.error(f"Repair {repair_name} failed: {e}")
                results[repair_name] = 'FAILED'

        # Save repair log
        log_file = f"/var/log/repair_log_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(log_file, 'w') as f:
            json.dump({
                'repair_session': {
                    'timestamp': datetime.now().isoformat(),
                    'dry_run': self.dry_run,
                    'results': results,
                    'actions': self.repair_log
                }
            }, f, indent=2)

        self.logger.info(f"Repair process completed. Log saved to: {log_file}")
        return results

# Usage example
if __name__ == "__main__":
    db_config = {
        'host': 'localhost',
        'database': 'semantic_text_processor',
        'user': 'your_user',
        'password': 'your_password'
    }

    # Run in dry-run mode first
    repair_manager = DataRepairManager(db_config, dry_run=True)
    results = repair_manager.run_all_repairs()

    print("Dry run results:")
    for repair, status in results.items():
        print(f"  {repair}: {status}")

    # Uncomment to run actual repairs
    # repair_manager = DataRepairManager(db_config, dry_run=False)
    # results = repair_manager.run_all_repairs()
```

### Emergency Repair Procedures

```bash
#!/bin/bash
# emergency_repair.sh

set -e

API_BASE="http://localhost:8080/api/v1"
BACKUP_DIR="/var/backup/emergency_$(date +%Y%m%d_%H%M%S)"

echo "=== EMERGENCY DATA REPAIR PROCEDURE ==="
echo "Backup directory: $BACKUP_DIR"
mkdir -p "$BACKUP_DIR"

# Function to create emergency backup
create_emergency_backup() {
    echo "Creating emergency backup..."

    # Backup current data state
    curl -s "$API_BASE/admin/export/chunks" > "$BACKUP_DIR/chunks_backup.json"
    curl -s "$API_BASE/admin/export/embeddings" > "$BACKUP_DIR/embeddings_backup.json"
    curl -s "$API_BASE/admin/export/templates" > "$BACKUP_DIR/templates_backup.json"

    echo "Emergency backup completed: $BACKUP_DIR"
}

# Function to repair critical issues immediately
repair_critical_issues() {
    echo "Repairing critical consistency issues..."

    # Stop application to prevent further corruption
    echo "Stopping application..."
    systemctl stop semantic-text-processor || echo "Warning: Could not stop service"

    # Run critical repairs
    python3 /usr/local/bin/data_repair.py --emergency --no-dry-run

    # Restart application
    echo "Restarting application..."
    systemctl start semantic-text-processor

    # Verify system health
    sleep 10
    HEALTH=$(curl -s "$API_BASE/health" | jq -r '.status // "unknown"')

    if [ "$HEALTH" = "healthy" ]; then
        echo "✅ Emergency repair completed successfully"
    else
        echo "❌ Emergency repair failed - system still unhealthy"
        exit 1
    fi
}

# Main emergency procedure
echo "WARNING: This will perform emergency repairs that may affect system availability."
read -p "Do you want to continue? (yes/no): " confirm

if [ "$confirm" = "yes" ]; then
    create_emergency_backup
    repair_critical_issues
    echo "Emergency repair procedure completed!"
else
    echo "Emergency repair cancelled."
    exit 0
fi
```

This comprehensive data consistency documentation provides all the tools and procedures needed to maintain data integrity in the Semantic Text Processor system. The approach is designed to be both proactive (preventing issues) and reactive (quickly resolving issues when they occur).