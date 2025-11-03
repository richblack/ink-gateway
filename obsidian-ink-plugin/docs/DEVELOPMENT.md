# Development Guide

This guide covers setting up the development environment and contributing to the Obsidian Ink Plugin.

## Prerequisites

- Node.js 16+ and npm
- Git
- Obsidian (for testing)
- TypeScript knowledge
- Familiarity with Obsidian plugin development

## Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/ink-gateway/obsidian-plugin.git
cd obsidian-plugin
```

### 2. Install Dependencies

```bash
npm install
```

### 3. Development Build

```bash
npm run dev
```

This starts the development build with file watching. The plugin will be rebuilt automatically when you make changes.

### 4. Link to Obsidian

Create a symbolic link from your Obsidian plugins directory to the development directory:

```bash
# Linux/Mac
ln -s /path/to/obsidian-plugin /path/to/vault/.obsidian/plugins/obsidian-ink-plugin

# Windows
mklink /D "C:\path\to\vault\.obsidian\plugins\obsidian-ink-plugin" "C:\path\to\obsidian-plugin"
```

### 5. Enable in Obsidian

1. Open Obsidian
2. Go to Settings > Community Plugins
3. Enable "Ink Gateway Plugin"

## Project Structure

```
obsidian-ink-plugin/
├── src/                    # Source code
│   ├── main.ts            # Main plugin class
│   ├── types/             # Type definitions
│   ├── interfaces/        # Interface definitions
│   ├── managers/          # Core managers (ContentManager, SearchManager, etc.)
│   ├── api/              # API client and related code
│   ├── ui/               # User interface components
│   └── utils/            # Utility functions
├── tests/                 # Test files
├── docs/                  # Documentation
├── manifest.json          # Plugin manifest
├── package.json          # Node.js dependencies
├── tsconfig.json         # TypeScript configuration
├── esbuild.config.mjs    # Build configuration
└── jest.config.js        # Test configuration
```

## Available Scripts

```bash
# Development
npm run dev          # Start development build with watching
npm run build        # Production build
npm run lint         # Run ESLint
npm run lint:fix     # Fix ESLint issues

# Testing
npm test             # Run all tests
npm run test:watch   # Run tests in watch mode

# Versioning
npm run version      # Bump version and update manifest
```

## Development Workflow

### 1. Making Changes

1. Create a feature branch: `git checkout -b feature/your-feature`
2. Make your changes
3. Test thoroughly
4. Commit with descriptive messages
5. Push and create a pull request

### 2. Testing

Always test your changes:

```bash
# Run unit tests
npm test

# Test in Obsidian
# 1. Build the plugin: npm run dev
# 2. Reload Obsidian
# 3. Test functionality manually
```

### 3. Code Style

- Use TypeScript strict mode
- Follow existing code patterns
- Add JSDoc comments for public APIs
- Use meaningful variable and function names
- Keep functions small and focused

### 4. Commit Messages

Use conventional commit format:

```
feat: add semantic search functionality
fix: resolve sync error handling
docs: update installation instructions
test: add content manager tests
```

## Architecture Overview

### Core Components

1. **Main Plugin Class** (`src/main.ts`)
   - Plugin lifecycle management
   - Component initialization
   - Event handling

2. **Managers** (`src/managers/`)
   - `ContentManager`: Content parsing and sync
   - `SearchManager`: Search functionality
   - `TemplateManager`: Template system
   - `AIManager`: AI chat integration

3. **API Client** (`src/api/`)
   - HTTP client for Ink-Gateway
   - Request/response handling
   - Error handling and retries

4. **UI Components** (`src/ui/`)
   - Search view
   - Chat view
   - Settings view

### Design Principles

- **Separation of Concerns**: Each manager handles specific functionality
- **Interface-Driven**: Use interfaces for testability and modularity
- **Error Handling**: Graceful error handling with user feedback
- **Performance**: Efficient caching and lazy loading
- **Offline Support**: Queue operations when offline

## Testing

### Unit Tests

Write unit tests for all core functionality:

```typescript
// Example test
describe('ContentManager', () => {
  it('should parse markdown content correctly', () => {
    const content = '# Title\n\nContent here';
    const result = contentManager.parseContent(content, 'test.md');
    expect(result.chunks).toHaveLength(2);
  });
});
```

### Integration Tests

Test component interactions:

```typescript
describe('Plugin Integration', () => {
  it('should sync content to Ink-Gateway', async () => {
    const file = createMockFile('# Test\n\nContent');
    await plugin.contentManager.handleContentChange(file);
    expect(mockApiClient.createChunk).toHaveBeenCalled();
  });
});
```

### Manual Testing

1. Test in different Obsidian versions
2. Test with various vault sizes
3. Test offline/online scenarios
4. Test error conditions

## API Integration

### Ink-Gateway API

The plugin communicates with Ink-Gateway via REST API:

```typescript
// Example API call
const chunk = await this.apiClient.createChunk({
  chunkId: generateId(),
  contents: 'Content here',
  tags: ['tag1', 'tag2'],
  // ... other fields
});
```

### Error Handling

Handle API errors gracefully:

```typescript
try {
  await this.apiClient.createChunk(chunk);
} catch (error) {
  if (error.type === ErrorType.NETWORK_ERROR) {
    // Queue for retry
    this.offlineManager.queueOperation(operation);
  } else {
    // Show user error
    new Notice('Failed to sync content');
  }
}
```

## UI Development

### Obsidian UI Components

Use Obsidian's built-in UI components:

```typescript
import { Modal, Setting } from 'obsidian';

class SearchModal extends Modal {
  onOpen() {
    const { contentEl } = this;
    
    new Setting(contentEl)
      .setName('Search Query')
      .addText(text => text
        .setPlaceholder('Enter search terms...')
        .onChange(value => this.query = value)
      );
  }
}
```

### Custom Components

Create reusable UI components:

```typescript
class SearchResultComponent {
  constructor(
    private container: HTMLElement,
    private result: SearchResultItem
  ) {}
  
  render() {
    this.container.createEl('div', {
      text: this.result.chunk.contents,
      cls: 'search-result-item'
    });
  }
}
```

## Performance Optimization

### Caching Strategy

- Cache search results
- Cache parsed content
- Cache API responses
- Implement TTL for cache entries

### Memory Management

- Clean up event listeners
- Clear caches periodically
- Monitor memory usage
- Implement lazy loading

### Network Optimization

- Batch API requests
- Implement request debouncing
- Use compression when possible
- Handle rate limiting

## Debugging

### Debug Mode

Enable debug mode in plugin settings to get detailed logs:

```typescript
if (this.settings.debugMode) {
  this.logger.debug('Processing content:', content);
}
```

### Console Debugging

Use browser developer tools:

1. Open Developer Console (`Ctrl/Cmd + Shift + I`)
2. Look for `[Ink Plugin]` messages
3. Set breakpoints in source code
4. Inspect network requests

### Common Issues

1. **Plugin not loading**: Check manifest.json and main.js
2. **API errors**: Verify Ink-Gateway connection
3. **Performance issues**: Check cache size and memory usage
4. **Sync problems**: Review offline queue and conflict resolution

## Contributing

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit pull request

### Code Review

All changes require code review:

- Code follows style guidelines
- Tests pass
- Documentation is updated
- No breaking changes (or properly documented)

### Release Process

1. Update version in `package.json` and `manifest.json`
2. Update `CHANGELOG.md`
3. Create release tag
4. Build and publish to Obsidian community plugins

## Resources

- [Obsidian Plugin Developer Docs](https://docs.obsidian.md/Plugins/Getting+started/Build+a+plugin)
- [Obsidian API Reference](https://docs.obsidian.md/Reference/TypeScript+API)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Jest Testing Framework](https://jestjs.io/docs/getting-started)

## Getting Help

- Check existing issues on GitHub
- Join the Obsidian Discord
- Review Obsidian plugin development resources
- Ask questions in discussions