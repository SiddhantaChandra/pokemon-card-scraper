import React, { useState } from 'react';
import { useScrapeStatus } from '../hooks/useScrapeStatus';
import { databaseAPI } from '../lib/api';

const ScraperControls = ({ className = '' }) => {
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [scrapeOptions, setScrapeOptions] = useState({
    inStockOnly: false,
    maxPages: 0
  });

  const {
    isRunning,
    currentPage,
    totalPages,
    progressPercentage,
    totalCardsInDatabase,
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

  const handleResetDatabase = async () => {
    if (!showResetConfirm) {
      setShowResetConfirm(true);
      return;
    }

    try {
      setIsResetting(true);
      await databaseAPI.reset();
      setShowResetConfirm(false);
      refresh();
    } catch (error) {
      console.error('Failed to reset database:', error);
      alert('Failed to reset database. Please try again.');
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className="space-y-3">
      {/* Compact Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Scraper Controls</h3>
        <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
          <span className={`w-2 h-2 rounded-full ${isRunning ? 'bg-green-500' : 'bg-gray-400'}`}></span>
          <span>{isRunning ? 'Running' : 'Ready'}</span>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-sm text-red-700 dark:text-red-300">
          {error.message}
        </div>
      )}

      {/* Compact Options & Controls */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Options */}
        <div className="space-y-2">
          <div className="flex items-center gap-4">
            <label className="flex items-center text-sm">
              <input
                type="checkbox"
                checked={scrapeOptions.inStockOnly}
                onChange={(e) => setScrapeOptions(prev => ({ ...prev, inStockOnly: e.target.checked }))}
                className="rounded mr-2"
                disabled={isRunning}
              />
              In-stock only
            </label>
            <div className="flex items-center gap-1 text-sm">
              <span>Max pages:</span>
              <input
                type="number"
                min="0"
                placeholder="0"
                value={scrapeOptions.maxPages || ''}
                onChange={(e) => setScrapeOptions(prev => ({ ...prev, maxPages: parseInt(e.target.value) || 0 }))}
                className="w-16 px-1 py-0.5 border rounded text-center text-sm"
                disabled={isRunning}
              />
            </div>
          </div>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-2">
          {canStart && (
            <button
              onClick={handleStartScraping}
              disabled={isStarting}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded font-medium text-sm transition-colors disabled:opacity-50"
            >
              {isStarting ? 'Starting...' : 'Start'}
            </button>
          )}

          {canStop && (
            <button
              onClick={handleStopScraping}
              disabled={isStopping}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded font-medium text-sm transition-colors disabled:opacity-50"
            >
              {isStopping ? 'Stopping...' : 'Stop'}
            </button>
          )}

          <button
            onClick={refresh}
            className="px-3 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded font-medium text-sm transition-colors"
            title="Refresh"
          >
            ↻
          </button>
        </div>
      </div>

      {/* Progress Bar (only when running) */}
      {isRunning && totalPages > 0 && (
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400">
            <span>Page {currentPage} of {totalPages}</span>
            <span>{progressPercentage}%</span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
            <div 
              className="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
              style={{ width: `${progressPercentage}%` }}
            ></div>
          </div>
        </div>
      )}

      {/* Compact Stats */}
      <div className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400 pt-2 border-t border-gray-200 dark:border-gray-700">
        <span>{totalCardsInDatabase.toLocaleString()} cards in database</span>
        <button
          onClick={handleResetDatabase}
          disabled={isResetting}
          className="px-3 py-1 bg-orange-600 hover:bg-orange-700 text-white rounded text-xs font-medium transition-colors disabled:opacity-50"
        >
          {showResetConfirm ? (isResetting ? 'Resetting...' : 'Confirm Reset') : 'Reset Database'}
        </button>
      </div>

      {/* Reset Confirmation */}
      {showResetConfirm && !isResetting && (
        <div className="p-2 bg-orange-50 dark:bg-orange-900/20 border border-orange-200 dark:border-orange-800 rounded text-sm">
          <p className="text-orange-800 dark:text-orange-300 mb-2">Are you sure? This will delete all scraped cards but preserve tracker data.</p>
          <div className="flex gap-2">
            <button
              onClick={() => setShowResetConfirm(false)}
              className="px-2 py-1 bg-gray-500 hover:bg-gray-600 text-white rounded text-xs"
            >
              Cancel
            </button>
            <button
              onClick={handleResetDatabase}
              className="px-2 py-1 bg-orange-600 hover:bg-orange-700 text-white rounded text-xs"
            >
              Yes, Reset Cards
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default ScraperControls; 