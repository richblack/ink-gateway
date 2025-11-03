#!/usr/bin/env node

/**
 * Comprehensive Test Runner
 * Runs all test suites and generates reports
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

class TestRunner {
    constructor() {
        this.results = {
            unit: null,
            integration: null,
            performance: null,
            load: null,
            coverage: null,
        };
        this.startTime = Date.now();
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: '📋',
            success: '✅',
            error: '❌',
            warning: '⚠️',
            performance: '⚡',
        }[type] || '📋';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    async runCommand(command, description) {
        this.log(`Running: ${description}`, 'info');
        
        try {
            const startTime = Date.now();
            const output = execSync(command, { 
                encoding: 'utf8',
                stdio: 'pipe',
                cwd: process.cwd()
            });
            const duration = Date.now() - startTime;
            
            this.log(`✓ ${description} completed in ${duration}ms`, 'success');
            return { success: true, output, duration };
        } catch (error) {
            this.log(`✗ ${description} failed: ${error.message}`, 'error');
            return { success: false, error: error.message, output: error.stdout };
        }
    }

    async runUnitTests() {
        this.log('Starting Unit Tests', 'info');
        const result = await this.runCommand(
            'npm run test:unit',
            'Unit Tests'
        );
        this.results.unit = result;
        return result;
    }

    async runIntegrationTests() {
        this.log('Starting Integration Tests', 'info');
        const result = await this.runCommand(
            'npm run test:integration',
            'Integration Tests'
        );
        this.results.integration = result;
        return result;
    }

    async runPerformanceTests() {
        this.log('Starting Performance Tests', 'performance');
        const result = await this.runCommand(
            'npm run test:performance',
            'Performance Tests'
        );
        this.results.performance = result;
        return result;
    }

    async runLoadTests() {
        this.log('Starting Load Tests', 'performance');
        const result = await this.runCommand(
            'npm run test:load',
            'Load Tests'
        );
        this.results.load = result;
        return result;
    }

    async generateCoverageReport() {
        this.log('Generating Coverage Report', 'info');
        const result = await this.runCommand(
            'npm run test:coverage',
            'Coverage Report Generation'
        );
        this.results.coverage = result;
        return result;
    }

    async runAllTests() {
        this.log('🚀 Starting Comprehensive Test Suite', 'info');
        
        // Run tests in sequence
        await this.runUnitTests();
        await this.runIntegrationTests();
        await this.runPerformanceTests();
        await this.runLoadTests();
        await this.generateCoverageReport();
        
        this.generateSummaryReport();
    }

    generateSummaryReport() {
        const totalDuration = Date.now() - this.startTime;
        
        this.log('📊 Test Suite Summary', 'info');
        console.log('=' * 50);
        
        const testTypes = ['unit', 'integration', 'performance', 'load', 'coverage'];
        let totalPassed = 0;
        let totalFailed = 0;
        
        testTypes.forEach(type => {
            const result = this.results[type];
            if (result) {
                const status = result.success ? '✅ PASSED' : '❌ FAILED';
                const duration = result.duration ? `(${result.duration}ms)` : '';
                console.log(`  ${type.toUpperCase().padEnd(12)} ${status} ${duration}`);
                
                if (result.success) totalPassed++;
                else totalFailed++;
            }
        });
        
        console.log('=' * 50);
        console.log(`Total Duration: ${totalDuration}ms`);
        console.log(`Passed: ${totalPassed}, Failed: ${totalFailed}`);
        
        // Generate detailed report file
        this.generateDetailedReport();
        
        // Exit with appropriate code
        process.exit(totalFailed > 0 ? 1 : 0);
    }

    generateDetailedReport() {
        const report = {
            timestamp: new Date().toISOString(),
            totalDuration: Date.now() - this.startTime,
            results: this.results,
            summary: {
                passed: Object.values(this.results).filter(r => r && r.success).length,
                failed: Object.values(this.results).filter(r => r && !r.success).length,
            }
        };
        
        const reportPath = path.join(process.cwd(), 'test-results.json');
        fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
        
        this.log(`Detailed report saved to: ${reportPath}`, 'info');
        
        // Generate HTML report if coverage exists
        if (fs.existsSync(path.join(process.cwd(), 'coverage'))) {
            this.log('Coverage report available at: coverage/lcov-report/index.html', 'info');
        }
    }

    async runSpecificTest(testType) {
        switch (testType) {
            case 'unit':
                return await this.runUnitTests();
            case 'integration':
                return await this.runIntegrationTests();
            case 'performance':
                return await this.runPerformanceTests();
            case 'load':
                return await this.runLoadTests();
            case 'coverage':
                return await this.generateCoverageReport();
            default:
                this.log(`Unknown test type: ${testType}`, 'error');
                return { success: false, error: 'Unknown test type' };
        }
    }
}

// CLI Interface
async function main() {
    const args = process.argv.slice(2);
    const testRunner = new TestRunner();
    
    if (args.length === 0) {
        // Run all tests
        await testRunner.runAllTests();
    } else {
        // Run specific test type
        const testType = args[0];
        const result = await testRunner.runSpecificTest(testType);
        
        if (result.success) {
            testRunner.log(`${testType} tests completed successfully`, 'success');
            process.exit(0);
        } else {
            testRunner.log(`${testType} tests failed`, 'error');
            process.exit(1);
        }
    }
}

// Handle uncaught errors
process.on('unhandledRejection', (error) => {
    console.error('❌ Unhandled rejection:', error);
    process.exit(1);
});

process.on('uncaughtException', (error) => {
    console.error('❌ Uncaught exception:', error);
    process.exit(1);
});

if (require.main === module) {
    main().catch(error => {
        console.error('❌ Test runner failed:', error);
        process.exit(1);
    });
}

module.exports = TestRunner;