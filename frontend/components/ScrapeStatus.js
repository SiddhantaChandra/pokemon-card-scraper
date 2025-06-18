import { useState } from 'react';
import { useScrapeStatus } from '../hooks/useScrapeStatus';

export default function ScrapeStatus({ className = '' }) {
  const [showDetails, setShowDetails] = useState(false);
  const [scrapeOptions, setScrapeOptions] = useState({
    inStockOnly: true,
    maxPages: 0 // 0 means all pages
  });

  const {
    isRunning,
    currentPage,
    totalPages,
    itemsScraped,
    progressPercentage,
    estimatedTimeRemaining,
    lastUpdatedText,
    isLoading,
    isStarting,
    isStopping,
    error,
    startScraping,
    stopScraping,
    refresh,
    canStart,
    canStop
  } = useScrapeStatus();

  const handleStartScraping = async () => {
    const result = await startScraping(scrapeOptions);
    if (!result.success) {
      alert(`Failed to start scraping: ${result.error}`);
    }
  };

  const handleStopScraping = async () => {
    const result = await stopScraping();
    if (!result.success) {
      alert(`Failed to stop scraping: ${result.error}`);
    }
  };

  if (isLoading) {
    return (
      <div className={`bg-white rounded-lg shadow-md border border-gray-200 p-4 ${className}`}>
        <div className="animate-pulse flex items-center space-x-4">
          <div className="rounded-full bg-gray-300 h-10 w-10"></div>
          <div className="flex-1 space-y-2">
            <div className="h-4 bg-gray-300 rounded w-3/4"></div>
            <div className="h-3 bg-gray-300 rounded w-1/2"></div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white rounded-lg shadow-md border border-gray-200 ${className}`}>
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            {/* Status Icon */}
            <div className={`w-3 h-3 rounded-full ${
              isRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
            }`}></div>
            
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                Scraping Status
              </h3>
              <p className="text-sm text-gray-500">
                {isRunning ? 'Currently scraping...' : 'Ready to scrape'}
              </p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center space-x-2">
            <button
              onClick={refresh}
              className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
              title="Refresh status"
            >
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>

            <button
              onClick={() => setShowDetails(!showDetails)}
              className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
              title={showDetails ? 'Hide details' : 'Show details'}
            >
              <svg className={`h-5 w-5 transform transition-transform ${showDetails ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          </div>
        </div>

        {/* Progress Bar */}
        {isRunning && totalPages > 0 && (
          <div className="mt-4">
            <div className="flex items-center justify-between text-sm text-gray-600 mb-2">
              <span>Page {currentPage} of {totalPages}</span>
              <span>{progressPercentage}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div 
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${progressPercentage}%` }}
              ></div>
            </div>
            {estimatedTimeRemaining && (
              <div className="text-xs text-gray-500 mt-1">
                Estimated time remaining: {estimatedTimeRemaining}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Details Section */}
      {showDetails && (
        <div className="p-4 bg-gray-50">
          {/* Statistics */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{itemsScraped}</div>
              <div className="text-xs text-gray-500">Items Scraped</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{currentPage}</div>
              <div className="text-xs text-gray-500">Current Page</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{totalPages}</div>
              <div className="text-xs text-gray-500">Total Pages</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{progressPercentage}%</div>
              <div className="text-xs text-gray-500">Progress</div>
            </div>
          </div>

          {/* Last Updated */}
          <div className="text-sm text-gray-500 mb-4">
            Last updated: {lastUpdatedText}
          </div>

          {/* Scrape Options (when not running) */}
          {!isRunning && (
            <div className="mb-4">
              <h4 className="text-sm font-medium text-gray-700 mb-2">Scrape Options</h4>
              <div className="space-y-2">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={scrapeOptions.inStockOnly}
                    onChange={(e) => setScrapeOptions(prev => ({ ...prev, inStockOnly: e.target.checked }))}
                    className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <span className="ml-2 text-sm text-gray-700">In-stock items only</span>
                </label>
                
                <div className="flex items-center space-x-2">
                  <label className="text-sm text-gray-700">Max pages:</label>
                  <input
                    type="number"
                    min="0"
                    placeholder="0 = all"
                    value={scrapeOptions.maxPages || ''}
                    onChange={(e) => setScrapeOptions(prev => ({ ...prev, maxPages: parseInt(e.target.value) || 0 }))}
                    className="w-20 px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                  />
                  <span className="text-xs text-gray-500">(0 = all pages)</span>
                </div>
              </div>
            </div>
          )}

          {/* Error Display */}
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
              <div className="flex">
                <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">Scraping Error</h3>
                  <p className="text-sm text-red-700 mt-1">{error.message}</p>
                </div>
              </div>
            </div>
          )}

          {/* Control Buttons */}
          <div className="flex items-center space-x-3">
            {canStart && (
              <button
                onClick={handleStartScraping}
                disabled={isStarting}
                className="flex items-center space-x-2 bg-green-600 hover:bg-green-700 disabled:bg-green-400 text-white px-4 py-2 rounded-md font-medium transition-colors duration-200"
              >
                {isStarting ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                    <span>Starting...</span>
                  </>
                ) : (
                  <>
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h1m4 0h1m-6 4h1m4 0h1m-6-8h1m4 0h1M7 21l4-16m2 16l4-16" />
                    </svg>
                    <span>Start Scraping</span>
                  </>
                )}
              </button>
            )}

            {canStop && (
              <button
                onClick={handleStopScraping}
                disabled={isStopping}
                className="flex items-center space-x-2 bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-4 py-2 rounded-md font-medium transition-colors duration-200"
              >
                {isStopping ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                    <span>Stopping...</span>
                  </>
                ) : (
                  <>
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10h6v4H9z" />
                    </svg>
                    <span>Stop Scraping</span>
                  </>
                )}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
} 