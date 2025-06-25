import { useScrapeStatus } from '../hooks/useScrapeStatus';

export default function ScrapeStatus({ className = '' }) {
  const {
    isRunning,
    currentPage,
    totalPages,
    itemsScraped,
    totalCardsInDatabase,
    progressPercentage,
    estimatedTimeRemaining,
    lastUpdatedText,
    isLoading,
    refresh
  } = useScrapeStatus();

  if (isLoading) {
    return (
      <div className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-3 ${className}`}>
        <div className="animate-pulse flex items-center space-x-3">
          <div className="rounded-full bg-gray-300 dark:bg-gray-600 h-6 w-6"></div>
          <div className="flex-1 space-y-1">
            <div className="h-3 bg-gray-300 dark:bg-gray-600 rounded w-3/4"></div>
            <div className="h-2 bg-gray-300 dark:bg-gray-600 rounded w-1/2"></div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-3 ${className}`}>
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${
            isRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
          }`}></div>
          <h4 className="font-medium text-gray-900 dark:text-white">
            {isRunning ? 'Scraping in Progress' : 'Scraper Status'}
          </h4>
        </div>
        
        <button
          onClick={refresh}
          className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          title="Refresh status"
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      {/* Progress Bar (when running) */}
      {isRunning && totalPages > 0 && (
        <div className="mb-3">
          <div className="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
            <span>Page {currentPage} of {totalPages}</span>
            <span>{progressPercentage}%</span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
            <div 
              className="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
              style={{ width: `${progressPercentage}%` }}
            ></div>
          </div>
          {estimatedTimeRemaining && (
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              ETA: {estimatedTimeRemaining}
            </div>
          )}
        </div>
      )}

      {/* Compact Stats Grid */}
      <div className="grid grid-cols-4 gap-3 mb-3">
        <div className="text-center">
          <div className="text-lg font-bold text-gray-900 dark:text-white">
            {totalCardsInDatabase.toLocaleString()}
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">Items</div>
        </div>
        <div className="text-center">
          <div className="text-lg font-bold text-gray-900 dark:text-white">
            {isRunning ? currentPage : 0}
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">Page</div>
        </div>
        <div className="text-center">
          <div className="text-lg font-bold text-gray-900 dark:text-white">
            {isRunning ? totalPages : 0}
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">Total</div>
        </div>
        <div className="text-center">
          <div className="text-lg font-bold text-gray-900 dark:text-white">
            {isRunning ? progressPercentage : 0}%
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">Done</div>
        </div>
      </div>

      {/* Last Updated */}
      <div className="text-xs text-gray-500 dark:text-gray-400 text-center">
        {lastUpdatedText}
      </div>
    </div>
  );
} 