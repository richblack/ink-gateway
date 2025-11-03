/**
 * Example usage of Document ID Management functionality
 * This file demonstrates how to use the document ID management features
 */

import { ContentManager } from '../ContentManager';
import { InkGatewayClient } from '../../api/InkGatewayClient';
import { VirtualDocumentContext, PaginationOptions } from '../../types';

// Example: Basic Document ID Management Usage
export async function documentIdManagementExample() {
  // Initialize the API client
  const apiClient = new InkGatewayClient('http://localhost:8080', 'your-api-key');
  
  // Initialize the content manager (simplified for example)
  const contentManager = new ContentManager(
    apiClient,
    console as any, // Simple logger
    { on: () => {}, off: () => {}, emit: () => {} } as any, // Simple event manager
    new Map() as any, // Simple cache manager
    { isOnline: () => true } as any, // Simple offline manager
    { vault: { read: () => '', getAbstractFileByPath: () => null } } as any // Simple app
  );

  // Example 1: Generate document ID for a physical file
  console.log('=== Example 1: Physical File Document ID ===');
  const filePath = 'notes/projects/my-project.md';
  const documentId = contentManager.generateDocumentId(filePath);
  console.log(`File path: ${filePath}`);
  console.log(`Document ID: ${documentId}`);
  console.log(`Is virtual: ${contentManager.isVirtualDocumentId(documentId)}`);
  console.log(`Extracted path: ${contentManager.extractFilePathFromDocumentId(documentId)}`);
  console.log();

  // Example 2: Generate virtual document ID
  console.log('=== Example 2: Virtual Document ID ===');
  const virtualContext: VirtualDocumentContext = {
    sourceType: 'remnote',
    contextId: 'page-123',
    pageTitle: 'My RemNote Page',
    metadata: {
      category: 'research',
      tags: ['important', 'project']
    }
  };
  
  const virtualDocId = contentManager.generateVirtualDocumentId(virtualContext);
  console.log(`Virtual context: ${JSON.stringify(virtualContext, null, 2)}`);
  console.log(`Virtual document ID: ${virtualDocId}`);
  console.log(`Is virtual: ${contentManager.isVirtualDocumentId(virtualDocId)}`);
  console.log(`Extracted path: ${contentManager.extractFilePathFromDocumentId(virtualDocId)}`);
  console.log();

  // Example 3: Retrieve chunks by document ID with pagination
  console.log('=== Example 3: Retrieve Chunks with Pagination ===');
  try {
    const paginationOptions: PaginationOptions = {
      page: 1,
      pageSize: 10,
      includeHierarchy: true,
      sortBy: 'position',
      sortOrder: 'asc'
    };

    console.log(`Retrieving chunks for document: ${documentId}`);
    console.log(`Pagination options: ${JSON.stringify(paginationOptions, null, 2)}`);
    
    // This would make an actual API call in a real scenario
    // const chunksResult = await contentManager.getChunksByDocumentId(documentId, paginationOptions);
    // console.log(`Retrieved ${chunksResult.chunks.length} chunks`);
    // console.log(`Total chunks: ${chunksResult.pagination.totalChunks}`);
    // console.log(`Current page: ${chunksResult.pagination.currentPage}/${chunksResult.pagination.totalPages}`);
    
    console.log('(API call would be made here in real usage)');
  } catch (error) {
    console.error('Error retrieving chunks:', error);
  }
  console.log();

  // Example 4: Create virtual document
  console.log('=== Example 4: Create Virtual Document ===');
  try {
    console.log('Creating virtual document...');
    
    // This would make an actual API call in a real scenario
    // const virtualDoc = await contentManager.createVirtualDocument(virtualContext);
    // console.log(`Created virtual document: ${virtualDoc.virtualDocumentId}`);
    // console.log(`Chunk IDs: ${virtualDoc.chunkIds.join(', ')}`);
    
    console.log('(API call would be made here in real usage)');
  } catch (error) {
    console.error('Error creating virtual document:', error);
  }
  console.log();

  // Example 5: Reconstruct document from chunks
  console.log('=== Example 5: Reconstruct Document ===');
  try {
    console.log(`Reconstructing document: ${documentId}`);
    
    // This would make an actual API call in a real scenario
    // const reconstructed = await contentManager.reconstructDocument(documentId);
    // console.log(`Reconstructed document with ${reconstructed.chunks.length} chunks`);
    // console.log(`Hierarchy nodes: ${reconstructed.hierarchy.length}`);
    // console.log(`Document metadata: ${JSON.stringify(reconstructed.metadata, null, 2)}`);
    
    console.log('(API call would be made here in real usage)');
  } catch (error) {
    console.error('Error reconstructing document:', error);
  }
  console.log();

  // Example 6: Update document scope
  console.log('=== Example 6: Update Document Scope ===');
  try {
    const chunkId = 'chunk-123';
    const newScope = 'virtual';
    
    console.log(`Updating chunk ${chunkId} to scope: ${newScope}`);
    
    // This would make an actual API call in a real scenario
    // await contentManager.updateDocumentScope(chunkId, documentId, newScope);
    // console.log('Document scope updated successfully');
    
    console.log('(API call would be made here in real usage)');
  } catch (error) {
    console.error('Error updating document scope:', error);
  }
  console.log();

  console.log('=== Document ID Management Examples Complete ===');
}

// Example: Working with different file types and paths
export function filePathExamples() {
  const contentManager = new ContentManager(
    {} as any, {} as any, {} as any, {} as any, {} as any, {} as any
  );

  console.log('=== File Path Examples ===');
  
  const testPaths = [
    'README.md',
    'docs/api/reference.md',
    'projects/2024/Q1/planning.md',
    'notes/daily/2024-01-15.md',
    'templates/meeting-notes.md',
    'archive/old-project/notes.md'
  ];

  testPaths.forEach(path => {
    const docId = contentManager.generateDocumentId(path);
    const extractedPath = contentManager.extractFilePathFromDocumentId(docId);
    const isVirtual = contentManager.isVirtualDocumentId(docId);
    
    console.log(`Original: ${path}`);
    console.log(`Document ID: ${docId}`);
    console.log(`Extracted: ${extractedPath}`);
    console.log(`Is Virtual: ${isVirtual}`);
    console.log(`Match: ${path === extractedPath ? '✓' : '✗'}`);
    console.log('---');
  });
}

// Example: Working with different virtual document types
export function virtualDocumentExamples() {
  const contentManager = new ContentManager(
    {} as any, {} as any, {} as any, {} as any, {} as any, {} as any
  );

  console.log('=== Virtual Document Examples ===');
  
  const virtualContexts: VirtualDocumentContext[] = [
    {
      sourceType: 'remnote',
      contextId: 'rem-page-123',
      pageTitle: 'Research Notes',
      metadata: { category: 'research' }
    },
    {
      sourceType: 'logseq',
      contextId: 'logseq-block-456',
      pageTitle: 'Daily Journal',
      metadata: { date: '2024-01-15' }
    },
    {
      sourceType: 'obsidian-template',
      contextId: 'template-meeting-notes',
      pageTitle: 'Meeting Notes Template',
      metadata: { type: 'template' }
    }
  ];

  virtualContexts.forEach(context => {
    const virtualId = contentManager.generateVirtualDocumentId(context);
    const isVirtual = contentManager.isVirtualDocumentId(virtualId);
    const extractedPath = contentManager.extractFilePathFromDocumentId(virtualId);
    
    console.log(`Source Type: ${context.sourceType}`);
    console.log(`Context ID: ${context.contextId}`);
    console.log(`Virtual Document ID: ${virtualId}`);
    console.log(`Is Virtual: ${isVirtual}`);
    console.log(`Extracted Path: ${extractedPath}`);
    console.log('---');
  });
}

// Run examples if this file is executed directly
if (require.main === module) {
  console.log('Running Document ID Management Examples...\n');
  
  filePathExamples();
  console.log('\n');
  
  virtualDocumentExamples();
  console.log('\n');
  
  // Note: documentIdManagementExample() requires actual API calls
  // Uncomment the line below to run it with a real API connection
  // documentIdManagementExample();
}