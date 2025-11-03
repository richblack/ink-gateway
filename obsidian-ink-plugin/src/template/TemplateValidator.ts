/**
 * Template Validator for validating template content and auto-fill mechanisms
 * Handles template content validation and automatic slot value population
 * Implements requirements 4.1, 4.2, 4.3, 4.6
 */

import { TFile, CachedMetadata } from 'obsidian';
import {
    Template,
    TemplateInstance,
    TemplateSlot,
    ValidationRule,
    PluginError,
    ErrorType
} from '../types';
import { ILogger } from '../interfaces';

export interface ValidationResult {
    valid: boolean;
    errors: ValidationError[];
    warnings: ValidationWarning[];
    suggestions: ValidationSuggestion[];
}

export interface ValidationError {
    type: 'required_slot_missing' | 'invalid_slot_value' | 'constraint_violation' | 'syntax_error';
    slotId?: string;
    slotName?: string;
    message: string;
    severity: 'error' | 'warning';
}

export interface ValidationWarning {
    type: 'optional_slot_empty' | 'deprecated_syntax' | 'performance_concern';
    slotId?: string;
    slotName?: string;
    message: string;
}

export interface ValidationSuggestion {
    type: 'auto_fill' | 'format_improvement' | 'content_enhancement';
    slotId?: string;
    slotName?: string;
    suggestedValue?: any;
    message: string;
}

export interface AutoFillResult {
    success: boolean;
    filledSlots: Record<string, any>;
    errors: string[];
    suggestions: ValidationSuggestion[];
}

export class TemplateValidator {
    private logger: ILogger;

    constructor(logger: ILogger) {
        this.logger = logger;
    }

    /**
     * Validate template instance against template definition
     */
    validateTemplateInstance(template: Template, instance: TemplateInstance): ValidationResult {
        try {
            this.logger.debug(`Validating template instance: ${instance.id}`);

            const result: ValidationResult = {
                valid: true,
                errors: [],
                warnings: [],
                suggestions: []
            };

            // Validate each slot
            template.slots.forEach(slot => {
                const slotValue = instance.slotValues[slot.id];
                const slotValidation = this.validateSlot(slot, slotValue);

                result.errors.push(...slotValidation.errors);
                result.warnings.push(...slotValidation.warnings);
                result.suggestions.push(...slotValidation.suggestions);
            });

            // Check for orphaned slot values (values without corresponding slots)
            Object.keys(instance.slotValues).forEach(slotId => {
                const slot = template.slots.find(s => s.id === slotId);
                if (!slot) {
                    result.warnings.push({
                        type: 'deprecated_syntax',
                        slotId,
                        message: `Slot value exists but slot definition not found: ${slotId}`
                    });
                }
            });

            result.valid = result.errors.filter(e => e.severity === 'error').length === 0;

            this.logger.debug(`Validation completed for instance: ${instance.id}, valid: ${result.valid}`);
            return result;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            this.logger.error(`Failed to validate template instance: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
            return {
                valid: false,
                errors: [{
                    type: 'syntax_error',
                    message: `Validation error: ${errorMessage}`,
                    severity: 'error'
                }],
                warnings: [],
                suggestions: []
            };
        }
    }

    /**
     * Auto-fill template slots with intelligent defaults
     */
    async autoFillTemplate(
        template: Template,
        instance: TemplateInstance,
        context?: {
            file?: TFile;
            metadata?: CachedMetadata;
            userPreferences?: Record<string, any>;
        }
    ): Promise<AutoFillResult> {
        try {
            this.logger.debug(`Auto-filling template: ${template.name} for instance: ${instance.id}`);

            const result: AutoFillResult = {
                success: true,
                filledSlots: { ...instance.slotValues },
                errors: [],
                suggestions: []
            };

            // Auto-fill each slot
            for (const slot of template.slots) {
                const currentValue = result.filledSlots[slot.id];

                // Skip if slot already has a value
                if (currentValue !== undefined && currentValue !== null && currentValue !== '') {
                    continue;
                }

                const autoFillValue = await this.generateAutoFillValue(slot, context);
                if (autoFillValue !== undefined) {
                    result.filledSlots[slot.id] = autoFillValue;

                    result.suggestions.push({
                        type: 'auto_fill',
                        slotId: slot.id,
                        slotName: slot.name,
                        suggestedValue: autoFillValue,
                        message: `Auto-filled ${slot.name} with: ${autoFillValue}`
                    });
                }
            }

            this.logger.debug(`Auto-fill completed for template: ${template.name}`);
            return result;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            this.logger.error(`Failed to auto-fill template: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
            return {
                success: false,
                filledSlots: instance.slotValues,
                errors: [errorMessage],
                suggestions: []
            };
        }
    }

    /**
     * Validate template content format and structure
     */
    validateTemplateContent(template: Template, content: string): ValidationResult {
        try {
            this.logger.debug(`Validating template content for: ${template.name}`);

            const result: ValidationResult = {
                valid: true,
                errors: [],
                warnings: [],
                suggestions: []
            };

            // Check if all template sections are present
            template.structure.sections.forEach(section => {
                const sectionPattern = new RegExp(`^##\\s+${this.escapeRegex(section.title)}`, 'm');
                if (!sectionPattern.test(content)) {
                    result.errors.push({
                        type: 'syntax_error',
                        message: `Missing required section: ${section.title}`,
                        severity: 'error'
                    });
                }
            });

            // Check for unfilled required slots
            template.slots.forEach(slot => {
                if (slot.required) {
                    const slotPattern = new RegExp(`\\{\\{${slot.name}(?::[^}]+)?\\}\\}`, 'g');
                    if (slotPattern.test(content)) {
                        result.errors.push({
                            type: 'required_slot_missing',
                            slotId: slot.id,
                            slotName: slot.name,
                            message: `Required slot not filled: ${slot.name}`,
                            severity: 'error'
                        });
                    }
                }
            });

            // Check for unfilled optional slots
            const unfilledSlots = content.match(/\{\{\w+(?::[^}]+)?\}\}/g);
            if (unfilledSlots && unfilledSlots.length > 0) {
                result.warnings.push({
                    type: 'optional_slot_empty',
                    message: `Unfilled optional slots found: ${unfilledSlots.join(', ')}`
                });
            }

            // Check for potential formatting improvements
            this.checkFormattingImprovements(content, result);

            result.valid = result.errors.filter(e => e.severity === 'error').length === 0;

            this.logger.debug(`Content validation completed for template: ${template.name}, valid: ${result.valid}`);
            return result;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            this.logger.error(`Failed to validate template content: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
            return {
                valid: false,
                errors: [{
                    type: 'syntax_error',
                    message: `Content validation error: ${errorMessage}`,
                    severity: 'error'
                }],
                warnings: [],
                suggestions: []
            };
        }
    }

    /**
     * Suggest improvements for template instance
     */
    suggestImprovements(template: Template, instance: TemplateInstance): ValidationSuggestion[] {
        const suggestions: ValidationSuggestion[] = [];

        try {
            template.slots.forEach(slot => {
                const value = instance.slotValues[slot.id];

                // Suggest improvements based on slot type and value
                if (value !== undefined && value !== null && value !== '') {
                    const slotSuggestions = this.generateSlotSuggestions(slot, value);
                    suggestions.push(...slotSuggestions);
                } else if (!slot.required) {
                    // Suggest filling optional slots
                    suggestions.push({
                        type: 'content_enhancement',
                        slotId: slot.id,
                        slotName: slot.name,
                        message: `Consider filling optional slot: ${slot.name}`
                    });
                }
            });

            this.logger.debug(`Generated ${suggestions.length} improvement suggestions`);
            return suggestions;

        } catch (error) {
            this.logger.error('Failed to generate improvement suggestions', error instanceof Error ? error : new Error(String(error)));
            return [];
        }
    }

    /**
     * Validate slot value against slot definition
     */
    private validateSlot(slot: TemplateSlot, value: any): {
        errors: ValidationError[];
        warnings: ValidationWarning[];
        suggestions: ValidationSuggestion[];
    } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];
        const suggestions: ValidationSuggestion[] = [];

        // Check required slots
        if (slot.required && (value === undefined || value === null || value === '')) {
            errors.push({
                type: 'required_slot_missing',
                slotId: slot.id,
                slotName: slot.name,
                message: `Required slot is empty: ${slot.name}`,
                severity: 'error'
            });
            return { errors, warnings, suggestions };
        }

        // Skip validation if value is empty and slot is not required
        if (value === undefined || value === null || value === '') {
            return { errors, warnings, suggestions };
        }

        // Validate slot value
        if (slot.validation) {
            const validationResult = this.validateSlotValue(value, slot.validation);
            if (!validationResult.valid) {
                errors.push({
                    type: 'invalid_slot_value',
                    slotId: slot.id,
                    slotName: slot.name,
                    message: `Invalid value for ${slot.name}: ${validationResult.error}`,
                    severity: 'error'
                });
            }
        }

        // Type-specific validation
        const typeValidation = this.validateSlotType(slot, value);
        errors.push(...typeValidation.errors);
        warnings.push(...typeValidation.warnings);
        suggestions.push(...typeValidation.suggestions);

        return { errors, warnings, suggestions };
    }

    private validateSlotValue(value: any, validation: ValidationRule): { valid: boolean; error?: string } {
        try {
            // Pattern validation
            if (validation.pattern && typeof value === 'string') {
                const regex = new RegExp(validation.pattern);
                if (!regex.test(value)) {
                    return { valid: false, error: `Value does not match required pattern: ${validation.pattern}` };
                }
            }

            // Length validation
            if (validation.minLength && typeof value === 'string') {
                if (value.length < validation.minLength) {
                    return { valid: false, error: `Value too short (minimum ${validation.minLength} characters)` };
                }
            }

            if (validation.maxLength && typeof value === 'string') {
                if (value.length > validation.maxLength) {
                    return { valid: false, error: `Value too long (maximum ${validation.maxLength} characters)` };
                }
            }

            // Custom validation
            if (validation.customValidator) {
                if (!validation.customValidator(value)) {
                    return { valid: false, error: 'Value failed custom validation' };
                }
            }

            return { valid: true };

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            return { valid: false, error: `Validation error: ${errorMessage}` };
        }
    }

    private validateSlotType(slot: TemplateSlot, value: any): {
        errors: ValidationError[];
        warnings: ValidationWarning[];
        suggestions: ValidationSuggestion[];
    } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];
        const suggestions: ValidationSuggestion[] = [];

        switch (slot.type) {
            case 'number':
                if (isNaN(Number(value))) {
                    errors.push({
                        type: 'invalid_slot_value',
                        slotId: slot.id,
                        slotName: slot.name,
                        message: `Expected number but got: ${typeof value}`,
                        severity: 'error'
                    });
                }
                break;

            case 'date':
                const date = new Date(value);
                if (isNaN(date.getTime())) {
                    errors.push({
                        type: 'invalid_slot_value',
                        slotId: slot.id,
                        slotName: slot.name,
                        message: `Invalid date format: ${value}`,
                        severity: 'error'
                    });
                }
                break;

            case 'link':
                if (typeof value === 'string') {
                    if (!value.startsWith('[[') || !value.endsWith(']]')) {
                        suggestions.push({
                            type: 'format_improvement',
                            slotId: slot.id,
                            slotName: slot.name,
                            suggestedValue: `[[${value}]]`,
                            message: `Consider formatting as Obsidian link: [[${value}]]`
                        });
                    }
                }
                break;

            case 'tag':
                if (typeof value === 'string') {
                    if (!value.startsWith('#')) {
                        suggestions.push({
                            type: 'format_improvement',
                            slotId: slot.id,
                            slotName: slot.name,
                            suggestedValue: `#${value}`,
                            message: `Consider formatting as tag: #${value}`
                        });
                    }
                }
                break;
        }

        return { errors, warnings, suggestions };
    }

    private async generateAutoFillValue(
        slot: TemplateSlot,
        context?: {
            file?: TFile;
            metadata?: CachedMetadata;
            userPreferences?: Record<string, any>;
        }
    ): Promise<any> {
        // Use default value if available
        if (slot.defaultValue !== undefined) {
            return slot.defaultValue;
        }

        // Generate contextual auto-fill values
        switch (slot.type) {
            case 'date':
                if (slot.name.toLowerCase().includes('created')) {
                    return context?.file?.stat.ctime ? new Date(context.file.stat.ctime) : new Date();
                }
                if (slot.name.toLowerCase().includes('modified')) {
                    return context?.file?.stat.mtime ? new Date(context.file.stat.mtime) : new Date();
                }
                return new Date();

            case 'text':
                if (slot.name.toLowerCase().includes('title') || slot.name.toLowerCase().includes('name')) {
                    return context?.file?.basename || 'Untitled';
                }
                if (slot.name.toLowerCase().includes('author')) {
                    return context?.userPreferences?.defaultAuthor || 'Unknown';
                }
                break;

            case 'tag':
                if (context?.metadata?.tags && context.metadata.tags.length > 0) {
                    const firstTag = context.metadata.tags[0];
                    const tagString = typeof firstTag === 'string' ? firstTag : (firstTag as any)?.tag || '';
                    return tagString.replace(/^#/, '');
                }
                break;

            case 'link':
                if (slot.name.toLowerCase().includes('parent') && context?.file) {
                    const parentFolder = context.file.parent?.name;
                    return parentFolder ? `[[${parentFolder}]]` : undefined;
                }
                break;
        }

        return undefined;
    }

    private generateSlotSuggestions(slot: TemplateSlot, value: any): ValidationSuggestion[] {
        const suggestions: ValidationSuggestion[] = [];

        // Type-specific suggestions
        switch (slot.type) {
            case 'date':
                if (typeof value === 'string' && !value.match(/^\d{4}-\d{2}-\d{2}$/)) {
                    const date = new Date(value);
                    if (!isNaN(date.getTime())) {
                        suggestions.push({
                            type: 'format_improvement',
                            slotId: slot.id,
                            slotName: slot.name,
                            suggestedValue: date.toISOString().split('T')[0],
                            message: `Consider using ISO date format: ${date.toISOString().split('T')[0]}`
                        });
                    }
                }
                break;

            case 'text':
                if (typeof value === 'string') {
                    // Suggest capitalization for titles
                    if (slot.name.toLowerCase().includes('title') && value !== value.charAt(0).toUpperCase() + value.slice(1)) {
                        suggestions.push({
                            type: 'format_improvement',
                            slotId: slot.id,
                            slotName: slot.name,
                            suggestedValue: value.charAt(0).toUpperCase() + value.slice(1),
                            message: `Consider capitalizing title: ${value.charAt(0).toUpperCase() + value.slice(1)}`
                        });
                    }
                }
                break;
        }

        return suggestions;
    }

    private checkFormattingImprovements(content: string, result: ValidationResult): void {
        // Check for consistent heading levels
        const headings = content.match(/^#+\s+.+$/gm) || [];
        if (headings.length > 0) {
            const levels = headings.map(h => h.match(/^#+/)?.[0].length || 0);
            const hasInconsistentLevels = levels.some((level, index) => {
                if (index === 0) return false;
                return level > levels[index - 1] + 1;
            });

            if (hasInconsistentLevels) {
                result.suggestions.push({
                    type: 'format_improvement',
                    message: 'Consider using consistent heading levels (no skipping levels)'
                });
            }
        }

        // Check for empty sections
        const sections = content.split(/^##\s+/m);
        sections.forEach((section, index) => {
            if (index > 0 && section.trim().split('\n').length < 3) {
                result.warnings.push({
                    type: 'optional_slot_empty',
                    message: `Section appears to be empty or very short`
                });
            }
        });
    }

    private escapeRegex(string: string): string {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }
}