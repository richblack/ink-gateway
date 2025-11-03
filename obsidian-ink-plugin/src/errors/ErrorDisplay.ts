/**
 * User-friendly error display system
 * Provides intuitive error messages and recovery suggestions
 */

import { Modal, App, Setting, Notice } from 'obsidian';
import { ErrorType, PluginError } from '../types';
import { ErrorLogEntry, ErrorSeverity } from './ErrorHandler';

// Error display options
export interface ErrorDisplayOptions {
  showDetails: boolean;
  showRecoveryActions: boolean;
  autoHide: boolean;
  hideDelay: number;
  allowReporting: boolean;
}

// Recovery action definition
export interface RecoveryAction {
  label: string;
  description: string;
  action: () => Promise<void>;
  primary?: boolean;
}

// Error display configuration
export interface ErrorDisplayConfig {
  title: string;
  message: string;
  details?: string;
  recoveryActions: RecoveryAction[];
  severity: ErrorSeverity;
  canRetry: boolean;
  canReport: boolean;
}

// Default display options
const DEFAULT_DISPLAY_OPTIONS: ErrorDisplayOptions = {
  showDetails: false,
  showRecoveryActions: true,
  autoHide: true,
  hideDelay: 5000,
  allowReporting: true
};

export class ErrorDisplayManager {
  private app: App;
  private options: ErrorDisplayOptions;
  private activeModals: Set<ErrorModal> = new Set();

  constructor(app: App, options: Partial<ErrorDisplayOptions> = {}) {
    this.app = app;
    this.options = { ...DEFAULT_DISPLAY_OPTIONS, ...options };
  }

  /**
   * Display error with appropriate UI based on severity
   */
  async displayError(entry: ErrorLogEntry, customConfig?: Partial<ErrorDisplayConfig>): Promise<void> {
    const config = this.buildDisplayConfig(entry, customConfig);

    switch (entry.severity) {
      case ErrorSeverity.CRITICAL:
        await this.showErrorModal(config);
        break;
      
      case ErrorSeverity.HIGH:
        await this.showErrorModal(config);
        break;
      
      case ErrorSeverity.MEDIUM:
        if (config.recoveryActions.length > 0) {
          await this.showErrorModal(config);
        } else {
          this.showErrorNotice(config);
        }
        break;
      
      case ErrorSeverity.LOW:
        this.showErrorNotice(config);
        break;
    }
  }

  /**
   * Show error in a modal dialog
   */
  private async showErrorModal(config: ErrorDisplayConfig): Promise<void> {
    const modal = new ErrorModal(this.app, config, this.options);
    this.activeModals.add(modal);
    
    modal.onClose = () => {
      this.activeModals.delete(modal);
    };
    
    modal.open();
  }

  /**
   * Show error as a notice
   */
  private showErrorNotice(config: ErrorDisplayConfig): void {
    const icon = this.getSeverityIcon(config.severity);
    const message = `${icon} ${config.message}`;
    
    const duration = this.options.autoHide ? this.options.hideDelay : 0;
    new Notice(message, duration);
  }

  /**
   * Build display configuration from error entry
   */
  private buildDisplayConfig(
    entry: ErrorLogEntry, 
    customConfig?: Partial<ErrorDisplayConfig>
  ): ErrorDisplayConfig {
    const baseConfig: ErrorDisplayConfig = {
      title: this.getErrorTitle(entry.error),
      message: entry.userMessage,
      details: this.formatErrorDetails(entry),
      recoveryActions: this.getRecoveryActions(entry),
      severity: entry.severity,
      canRetry: entry.error.recoverable,
      canReport: this.options.allowReporting
    };

    return { ...baseConfig, ...customConfig };
  }

  /**
   * Get appropriate title for error
   */
  private getErrorTitle(error: PluginError): string {
    switch (error.type) {
      case ErrorType.NETWORK_ERROR:
        return 'Connection Problem';
      case ErrorType.API_ERROR:
        return 'Server Error';
      case ErrorType.SYNC_ERROR:
        return 'Sync Issue';
      case ErrorType.PARSING_ERROR:
        return 'Content Processing Error';
      case ErrorType.VALIDATION_ERROR:
        return 'Invalid Input';
      default:
        return 'Unexpected Error';
    }
  }

  /**
   * Format error details for display
   */
  private formatErrorDetails(entry: ErrorLogEntry): string {
    const details = [];
    
    details.push(`Error Code: ${entry.error.code}`);
    details.push(`Component: ${entry.context.component}`);
    details.push(`Operation: ${entry.context.operation}`);
    details.push(`Time: ${entry.context.timestamp.toLocaleString()}`);
    
    if (entry.error.details) {
      details.push(`Details: ${JSON.stringify(entry.error.details, null, 2)}`);
    }
    
    return details.join('\n');
  }

  /**
   * Get recovery actions for error
   */
  private getRecoveryActions(entry: ErrorLogEntry): RecoveryAction[] {
    const actions: RecoveryAction[] = [];

    // Common retry action for recoverable errors
    if (entry.error.recoverable) {
      actions.push({
        label: 'Retry',
        description: 'Try the operation again',
        action: async () => {
          // This would need to be implemented by the calling component
          console.log('Retry action triggered for error:', entry.id);
        },
        primary: true
      });
    }

    // Specific actions based on error type
    switch (entry.error.type) {
      case ErrorType.NETWORK_ERROR:
        actions.push({
          label: 'Check Connection',
          description: 'Verify your internet connection and server status',
          action: async () => {
            // Could implement connection test here
            new Notice('Please check your internet connection and try again');
          }
        });
        break;

      case ErrorType.API_ERROR:
        if (entry.error.code === 'HTTP_401') {
          actions.push({
            label: 'Update API Key',
            description: 'Open settings to update your API key',
            action: async () => {
              // This would open the settings modal
              console.log('Opening settings for API key update');
            }
          });
        }
        break;

      case ErrorType.SYNC_ERROR:
        actions.push({
          label: 'Force Sync',
          description: 'Force a complete synchronization',
          action: async () => {
            console.log('Force sync triggered');
          }
        });
        break;

      case ErrorType.PARSING_ERROR:
        actions.push({
          label: 'View Content',
          description: 'Review the content that failed to parse',
          action: async () => {
            console.log('Opening content for review');
          }
        });
        break;
    }

    // Always add help action
    actions.push({
      label: 'Get Help',
      description: 'View troubleshooting guide',
      action: async () => {
        // Could open documentation or support page
        new Notice('Opening troubleshooting guide...');
      }
    });

    return actions;
  }

  /**
   * Get icon for severity level
   */
  private getSeverityIcon(severity: ErrorSeverity): string {
    switch (severity) {
      case ErrorSeverity.CRITICAL:
        return '🚨';
      case ErrorSeverity.HIGH:
        return '⚠️';
      case ErrorSeverity.MEDIUM:
        return '⚠️';
      case ErrorSeverity.LOW:
        return 'ℹ️';
      default:
        return '❓';
    }
  }

  /**
   * Close all active error modals
   */
  closeAllModals(): void {
    this.activeModals.forEach(modal => modal.close());
    this.activeModals.clear();
  }

  /**
   * Update display options
   */
  updateOptions(options: Partial<ErrorDisplayOptions>): void {
    this.options = { ...this.options, ...options };
  }
}

/**
 * Error modal for detailed error display
 */
class ErrorModal extends Modal {
  private config: ErrorDisplayConfig;
  private options: ErrorDisplayOptions;

  constructor(app: App, config: ErrorDisplayConfig, options: ErrorDisplayOptions) {
    super(app);
    this.config = config;
    this.options = options;
  }

  onOpen() {
    const { contentEl } = this;
    contentEl.empty();

    // Title
    contentEl.createEl('h2', { text: this.config.title });

    // Message
    const messageEl = contentEl.createEl('p', { 
      text: this.config.message,
      cls: 'error-message'
    });

    // Add severity styling
    messageEl.addClass(`error-severity-${this.config.severity}`);

    // Details section (collapsible)
    if (this.options.showDetails && this.config.details) {
      const detailsContainer = contentEl.createEl('details');
      detailsContainer.createEl('summary', { text: 'Technical Details' });
      detailsContainer.createEl('pre', { 
        text: this.config.details,
        cls: 'error-details'
      });
    }

    // Recovery actions
    if (this.options.showRecoveryActions && this.config.recoveryActions.length > 0) {
      const actionsContainer = contentEl.createEl('div', { cls: 'error-actions' });
      actionsContainer.createEl('h3', { text: 'What can you do?' });

      this.config.recoveryActions.forEach(action => {
        new Setting(actionsContainer)
          .setName(action.label)
          .setDesc(action.description)
          .addButton(btn => {
            btn.setButtonText(action.label);
            if (action.primary) {
              btn.setCta();
            }
            btn.onClick(async () => {
              try {
                await action.action();
                this.close();
              } catch (error) {
                console.error('Recovery action failed:', error);
                new Notice('Recovery action failed. Please try again.');
              }
            });
          });
      });
    }

    // Report error option
    if (this.config.canReport) {
      const reportContainer = contentEl.createEl('div', { cls: 'error-report' });
      new Setting(reportContainer)
        .setName('Report this error')
        .setDesc('Help improve the plugin by reporting this error')
        .addButton(btn => {
          btn.setButtonText('Report');
          btn.onClick(() => {
            this.reportError();
          });
        });
    }

    // Close button
    const buttonContainer = contentEl.createEl('div', { cls: 'error-modal-buttons' });
    const closeBtn = buttonContainer.createEl('button', { 
      text: 'Close',
      cls: 'mod-cta'
    });
    closeBtn.onclick = () => this.close();

    // Auto-hide for low severity errors
    if (this.config.severity === ErrorSeverity.LOW && this.options.autoHide) {
      setTimeout(() => {
        this.close();
      }, this.options.hideDelay);
    }
  }

  private reportError(): void {
    // Create error report
    const report = {
      title: this.config.title,
      message: this.config.message,
      details: this.config.details,
      severity: this.config.severity,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      pluginVersion: '1.0.0' // This should come from plugin manifest
    };

    // Copy to clipboard
    navigator.clipboard.writeText(JSON.stringify(report, null, 2)).then(() => {
      new Notice('Error report copied to clipboard. Please paste it in your bug report.');
    }).catch(() => {
      new Notice('Failed to copy error report. Please manually copy the details.');
    });

    this.close();
  }

  onClose() {
    const { contentEl } = this;
    contentEl.empty();
  }
}

// CSS styles for error display (to be added to the plugin's CSS)
export const ERROR_DISPLAY_STYLES = `
.error-message {
  margin: 1em 0;
  padding: 0.5em;
  border-radius: 4px;
}

.error-severity-critical {
  background-color: var(--background-modifier-error);
  border-left: 4px solid var(--text-error);
}

.error-severity-high {
  background-color: var(--background-modifier-error);
  border-left: 4px solid var(--text-warning);
}

.error-severity-medium {
  background-color: var(--background-modifier-border);
  border-left: 4px solid var(--text-warning);
}

.error-severity-low {
  background-color: var(--background-modifier-border);
  border-left: 4px solid var(--text-muted);
}

.error-details {
  background-color: var(--background-primary-alt);
  padding: 1em;
  border-radius: 4px;
  font-family: var(--font-monospace);
  font-size: 0.9em;
  overflow-x: auto;
  white-space: pre-wrap;
}

.error-actions {
  margin-top: 1.5em;
  padding-top: 1em;
  border-top: 1px solid var(--background-modifier-border);
}

.error-report {
  margin-top: 1em;
  padding-top: 1em;
  border-top: 1px solid var(--background-modifier-border);
}

.error-modal-buttons {
  margin-top: 1.5em;
  text-align: right;
}

.error-modal-buttons button {
  margin-left: 0.5em;
}
`;

// Global error display manager instance
let globalErrorDisplayManager: ErrorDisplayManager | null = null;

export function initializeErrorDisplay(app: App, options?: Partial<ErrorDisplayOptions>): ErrorDisplayManager {
  globalErrorDisplayManager = new ErrorDisplayManager(app, options);
  return globalErrorDisplayManager;
}

export function getErrorDisplayManager(): ErrorDisplayManager | null {
  return globalErrorDisplayManager;
}