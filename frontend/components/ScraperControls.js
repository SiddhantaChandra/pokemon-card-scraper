import React, { useState } from 'react';
import { useScrapeStatus } from '../hooks/useScrapeStatus';
import { databaseAPI } from '../lib/api';
import ProgressBar from './ProgressBar';

const ScraperControls = ({ className = '' }) => {
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [scrapeOptions, setScrapeOptions] = useState({
    inStockOnly: false,
    maxPages: 0
  });

  const {
    isRunning,
    isPaused,
    currentPage,
    totalPages,
    itemsScraped,
    cardsPerMinute,
    estimatedTimeRemainingBackend,
    progressPercentage,
    totalCardsInDatabase,
    isStarting,
    isStopping,
    isPausing,
    isResuming,
    error,
    startScraping,
    stopScraping,
    pauseScraping,
    resumeScraping,
    restartScraping,
    refresh,
    canStart,
    canStop,
    canPause,
    canResume,
    canRestart,
    startTime
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

  const handlePauseScraping = async () => {
    const result = await pauseScraping();
    if (!result.success) {
      alert(`Failed to pause scraping: ${result.error}`);
    }
  };

  const handleResumeScraping = async () => {
    const result = await resumeScraping();
    if (!result.success) {
      alert(`Failed to resume scraping: ${result.error}`);
    }
  };

  const handleResetDatabase = async () => {
    if (!showResetConfirm) {
      setShowResetConfirm(true);
      return;
    }

    try {
      setIsResetting(true);
      await databaseAPI.resetDatabase();
      setShowResetConfirm(false);
      // Refresh stats after reset
      refresh();
    } catch (error) {
      console.error('Failed to reset database:', error);
      alert('Failed to reset database. Please try again.');
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className="w-full max-w-6xl mx-auto space-y-4">
      {/* Control Panel Header */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 overflow-hidden">
        <div className="bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 px-4 py-3">
          <h2 className="text-xl font-bold text-white">Pokemon Card Scraper</h2>
          <p className="text-indigo-100 mt-0.5 text-sm">Advanced web scraping with real-time progress tracking</p>
        </div>
        
        <div className="p-4">
          {/* Scrape Options (when not running) */}
          {!isRunning && (
            <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <h4 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">Scrape Options</h4>
              <div className="space-y-2">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={scrapeOptions.inStockOnly}
                    onChange={(e) => setScrapeOptions(prev => ({ ...prev, inStockOnly: e.target.checked }))}
                    className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:bg-gray-700"
                  />
                  <span className="ml-2 text-sm text-gray-700 dark:text-gray-300">In-stock items only</span>
                </label>
                
                <div className="flex items-center space-x-2">
                  <label className="text-sm text-gray-700 dark:text-gray-300 font-medium">Max pages:</label>
                  <input
                    type="number"
                    min="0"
                    placeholder="0 = all"
                    value={scrapeOptions.maxPages || ''}
                    onChange={(e) => setScrapeOptions(prev => ({ ...prev, maxPages: parseInt(e.target.value) || 0 }))}
                    className="w-20 px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
                  />
                  <span className="text-xs text-gray-500 dark:text-gray-400">(0 = all pages)</span>
                </div>
              </div>
            </div>
          )}

          {/* Error Display */}
          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <div className="flex">
                <svg className="h-4 w-4 text-red-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
                <div className="ml-2">
                  <h3 className="text-sm font-medium text-red-800 dark:text-red-400">Error</h3>
                  <p className="text-sm text-red-700 dark:text-red-300 mt-0.5">{error.message}</p>
                </div>
              </div>
            </div>
          )}

          {/* Control Buttons */}
          <div className="flex flex-wrap items-center gap-3 mb-4">
            {/* Start Button */}
            {canStart && (
              <button
                onClick={handleStartScraping}
                disabled={isStarting}
                className={`
                  inline-flex items-center px-6 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm
                  transition-all duration-200 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2
                  ${!isStarting
                    ? 'text-white bg-gradient-to-r from-green-500 to-emerald-600 hover:from-green-600 hover:to-emerald-700 focus:ring-green-500'
                    : 'text-gray-400 bg-gray-100 dark:bg-gray-700 cursor-not-allowed'
                  }
                `}
              >
                {isStarting ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-400 mr-2"></div>
                    <span>Starting...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h1m4 0h1m-6 4h1m4 0h1m-6-8h1m4 0h1M9 6h1m4 0h1" />
                    </svg>
                    <span>Start Scraping</span>
                  </>
                )}
              </button>
            )}

            {/* Pause Button */}
            {canPause && (
              <button
                onClick={handlePauseScraping}
                disabled={isPausing}
                className="
                  inline-flex items-center px-6 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm
                  text-white bg-gradient-to-r from-yellow-500 to-orange-500 hover:from-yellow-600 hover:to-orange-600
                  transition-all duration-200 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-yellow-500 focus:ring-offset-2
                "
              >
                {isPausing ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    <span>Pausing...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6" />
                    </svg>
                    <span>Pause</span>
                  </>
                )}
              </button>
            )}

            {/* Resume Button */}
            {canResume && (
              <button
                onClick={handleResumeScraping}
                disabled={isResuming}
                className="
                  inline-flex items-center px-6 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm
                  text-white bg-gradient-to-r from-blue-500 to-cyan-600 hover:from-blue-600 hover:to-cyan-700
                  transition-all duration-200 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                "
              >
                {isResuming ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    <span>Resuming...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h1m4 0h1m-6 4h1m4 0h1m-6-8h1m4 0h1M9 6h1m4 0h1" />
                    </svg>
                    <span>Resume</span>
                  </>
                )}
              </button>
            )}

            {/* Stop Button */}
            {canStop && (
              <button
                onClick={handleStopScraping}
                disabled={isStopping}
                className="
                  inline-flex items-center px-6 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm
                  text-white bg-gradient-to-r from-red-500 to-pink-600 hover:from-red-600 hover:to-pink-700
                  transition-all duration-200 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2
                "
              >
                {isStopping ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    <span>Stopping...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10h6v4H9z" />
                    </svg>
                    <span>Stop</span>
                  </>
                )}
              </button>
            )}

            {/* Restart Button */}
            {canRestart && isRunning && (
              <button
                onClick={handleRestartScraping}
                disabled={isStarting}
                className="
                  inline-flex items-center px-6 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm
                  text-white bg-gradient-to-r from-purple-500 to-indigo-600 hover:from-purple-600 hover:to-indigo-700
                  transition-all duration-200 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2
                "
              >
                {isStarting ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    <span>Restarting...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    <span>Restart</span>
                  </>
                )}
              </button>
            )}
          </div>

          {/* Status Message */}
          <div className="text-center">
            <div className={`inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium ${
              isRunning 
                ? isPaused 
                  ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400 border border-yellow-200 dark:border-yellow-800' 
                  : 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400 border border-green-200 dark:border-green-800'
                : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300 border border-gray-200 dark:border-gray-600'
            }`}>
              <div className={`w-2 h-2 rounded-full mr-2 ${
                isRunning 
                  ? isPaused 
                    ? 'bg-yellow-400 animate-pulse' 
                    : 'bg-green-400 animate-pulse'
                  : 'bg-gray-400'
              }`}></div>
              {isRunning 
                ? isPaused 
                  ? 'Scraping Paused' 
                  : 'Scraping Active'
                : 'Ready to Scrape'
              }
            </div>
          </div>
        </div>
      </div>

      {/* Progress Display - Only show when scraping */}
      {isRunning && (
        <ProgressBar
          progress={progressPercentage}
          cardsScraped={itemsScraped}
          totalPages={totalPages}
          currentPage={currentPage}
          cardsPerMinute={cardsPerMinute}
          estimatedTimeRemaining={estimatedTimeRemainingBackend}
          isPaused={isPaused}
          startTime={startTime}
        />
      )}

      {/* Quick Stats Summary - Show when not running */}
      {!isRunning && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-4">
          <h3 className="text-lg font-semibold text-gray-800 dark:text-white mb-3 text-center">Ready to Start</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold text-gray-400 dark:text-gray-500">0</p>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">Cards to Scrape</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-400 dark:text-gray-500">0</p>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">Pages to Process</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-400 dark:text-gray-500">0%</p>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">Progress</p>
            </div>
          </div>
          <div className="mt-4 text-center">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Click "Start Scraping" to begin collecting Pokemon card data with real-time progress tracking.
            </p>
          </div>
        </div>
      )}

      {/* Database Management - Show at bottom */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-4">
        <h4 className="text-lg font-semibold text-gray-800 dark:text-white mb-3">Database Management</h4>
        
        {!isRunning && totalCardsInDatabase > 0 && (
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600 dark:text-gray-300">
              {totalCardsInDatabase.toLocaleString()} cards in database
            </div>

            {!showResetConfirm ? (
              <button
                onClick={handleResetDatabase}
                disabled={isResetting}
                className="flex items-center space-x-2 bg-orange-600 hover:bg-orange-700 disabled:bg-orange-400 text-white px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 shadow-sm hover:shadow-md"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <span>Reset Database</span>
              </button>
            ) : (
              <div className="flex items-center space-x-2">
                <span className="text-sm text-orange-600 dark:text-orange-400 font-medium">
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
                  className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors duration-200"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>
        )}

        {totalCardsInDatabase === 0 && !isRunning && (
          <div className="text-sm text-gray-500 dark:text-gray-400 italic text-center">
            Database is empty. Start scraping to populate it.
          </div>
        )}

        {isRunning && (
          <div className="text-sm text-gray-500 dark:text-gray-400 italic text-center">
            Database operations are disabled while scraping is in progress.
          </div>
        )}
      </div>
    </div>
  );
};

export default ScraperControls; 