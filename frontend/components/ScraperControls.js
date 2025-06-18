import { useState } from 'react';
import { useScrapeStatus } from '../hooks/useScrapeStatus';
import { databaseAPI } from '../lib/api';

export default function ScraperControls({ className = '' }) {
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [scrapeOptions, setScrapeOptions] = useState({
    inStockOnly: true,
    maxPages: 0 // 0 means all pages
  });

  const {
    isRunning,
    totalCardsInDatabase,
    isStarting,
    isStopping,
    error,
    startScraping,
    stopScraping,
    restartScraping,
    refresh,
    canStart,
    canStop,
    canRestart
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

  const handleRestartScraping = async () => {
    const result = await restartScraping(scrapeOptions);
    if (!result.success) {
      alert(`Failed to restart scraping: ${result.error}`);
    }
  };

  const handleResetDatabase = async () => {
    if (!showResetConfirm) {
      setShowResetConfirm(true);
      return;
    }

    setIsResetting(true);
    try {
      const response = await databaseAPI.reset();
      alert(`Database reset successful! ${response.data.message}`);
      setShowResetConfirm(false);
      // Refresh stats after reset
      refresh();
    } catch (error) {
      console.error('Failed to reset database:', error);
      alert(`Failed to reset database: ${error.response?.data?.message || error.message}`);
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className={`bg-white rounded-lg shadow-md border border-gray-200 p-6 ${className}`}>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-xl font-bold text-gray-900">Scraper Controls</h3>
          <p className="text-gray-600 text-sm mt-1">
            Manage scraping operations and database
          </p>
        </div>
        <div className={`w-3 h-3 rounded-full ${
          isRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
        }`}></div>
      </div>

      {/* Scrape Options (when not running) */}
      {!isRunning && (
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <h4 className="text-sm font-semibold text-gray-700 mb-3">Scrape Options</h4>
          <div className="space-y-3">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={scrapeOptions.inStockOnly}
                onChange={(e) => setScrapeOptions(prev => ({ ...prev, inStockOnly: e.target.checked }))}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700">In-stock items only</span>
            </label>
            
            <div className="flex items-center space-x-3">
              <label className="text-sm text-gray-700 font-medium">Max pages:</label>
              <input
                type="number"
                min="0"
                placeholder="0 = all"
                value={scrapeOptions.maxPages || ''}
                onChange={(e) => setScrapeOptions(prev => ({ ...prev, maxPages: parseInt(e.target.value) || 0 }))}
                className="w-24 px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <span className="text-xs text-gray-500">(0 = all pages)</span>
            </div>
          </div>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
          <div className="flex">
            <svg className="h-5 w-5 text-red-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error</h3>
              <p className="text-sm text-red-700 mt-1">{error.message}</p>
            </div>
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="space-y-4">
        {/* Scraper Control Buttons */}
        <div className="flex flex-wrap gap-3">
          {canStart && (
            <button
              onClick={handleStartScraping}
              disabled={isStarting}
              className="flex items-center space-x-2 bg-green-600 hover:bg-green-700 disabled:bg-green-400 text-white px-4 py-2 rounded-lg font-medium transition-all duration-200 shadow-sm hover:shadow-md"
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
              className="flex items-center space-x-2 bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-4 py-2 rounded-lg font-medium transition-all duration-200 shadow-sm hover:shadow-md"
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

          {canRestart && (
            <button
              onClick={handleRestartScraping}
              disabled={isStarting}
              className="flex items-center space-x-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white px-4 py-2 rounded-lg font-medium transition-all duration-200 shadow-sm hover:shadow-md"
            >
              {isStarting ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                  <span>Restarting...</span>
                </>
              ) : (
                <>
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  <span>Restart Scraping</span>
                </>
              )}
            </button>
          )}
        </div>

        {/* Database Management */}
        <div className="pt-4 border-t border-gray-200">
          <h4 className="text-sm font-semibold text-gray-700 mb-3">Database Management</h4>
          
          {!isRunning && totalCardsInDatabase > 0 && (
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600">
                {totalCardsInDatabase.toLocaleString()} cards in database
              </div>
              
              {!showResetConfirm ? (
                <button
                  onClick={handleResetDatabase}
                  disabled={isResetting}
                  className="flex items-center space-x-2 bg-orange-600 hover:bg-orange-700 disabled:bg-orange-400 text-white px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 shadow-sm hover:shadow-md"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  <span>Reset Database</span>
                </button>
              ) : (
                <div className="flex items-center space-x-2">
                  <span className="text-sm text-orange-600 font-medium">
                    Delete all {totalCardsInDatabase.toLocaleString()} cards?
                  </span>
                  <button
                    onClick={handleResetDatabase}
                    disabled={isResetting}
                    className="flex items-center space-x-1 bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200"
                  >
                    {isResetting ? (
                      <>
                        <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white"></div>
                        <span>Deleting...</span>
                      </>
                    ) : (
                      <span>Confirm Delete</span>
                    )}
                  </button>
                  <button
                    onClick={() => setShowResetConfirm(false)}
                    disabled={isResetting}
                    className="px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50 transition-colors duration-200"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>
          )}

          {totalCardsInDatabase === 0 && !isRunning && (
            <div className="text-sm text-gray-500 italic">
              Database is empty. Start scraping to populate it.
            </div>
          )}

          {isRunning && (
            <div className="text-sm text-gray-500 italic">
              Database operations are disabled while scraping is in progress.
            </div>
          )}
        </div>
      </div>
    </div>
  );
} 