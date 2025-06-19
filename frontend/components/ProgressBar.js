import React from 'react';

const ProgressBar = ({ 
  progress, 
  cardsScraped, 
  totalPages, 
  currentPage, 
  cardsPerMinute, 
  estimatedTimeRemaining,
  isPaused,
  startTime 
}) => {
  // Helper function to format time duration
  const formatDuration = (milliseconds) => {
    if (!milliseconds || milliseconds <= 0) return 'Calculating...';
    
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    
    if (days > 0) return `${days}d ${hours % 24}h ${minutes % 60}m`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
  };

  // Helper function to format elapsed time
  const formatElapsedTime = (startTime) => {
    if (!startTime) return 'N/A';
    const elapsed = Date.now() - new Date(startTime).getTime();
    return formatDuration(elapsed);
  };

  // Helper function to format numbers with commas
  const formatNumber = (num) => {
    return num?.toLocaleString() || '0';
  };

  return (
    <div className="w-full max-w-6xl mx-auto space-y-6">
      {/* Main Progress Bar */}
      <div className="bg-white rounded-2xl shadow-lg border border-gray-100 overflow-hidden">
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-xl font-bold text-gray-800 flex items-center gap-2">
              <div className={`w-3 h-3 rounded-full ${isPaused ? 'bg-yellow-400' : 'bg-green-400'} animate-pulse`}></div>
              {isPaused ? 'Scraping Paused' : 'Scraping in Progress'}
            </h3>
            <div className="text-sm text-gray-500">
              {currentPage ? `Page ${formatNumber(currentPage)}` : ''} 
              {totalPages ? ` of ${formatNumber(totalPages)}` : ''}
            </div>
          </div>

          {/* Progress Bar */}
          <div className="relative">
            <div className="w-full bg-gray-200 rounded-full h-4 mb-4 overflow-hidden">
              <div 
                className={`h-full rounded-full transition-all duration-1000 ease-out ${
                  isPaused 
                    ? 'bg-gradient-to-r from-yellow-400 to-orange-400' 
                    : 'bg-gradient-to-r from-blue-500 via-purple-500 to-indigo-600'
                }`}
                style={{ width: `${Math.min(progress || 0, 100)}%` }}
              >
                <div className="h-full w-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-pulse"></div>
              </div>
            </div>
            
            {/* Progress Percentage Overlay */}
            <div className="absolute top-0 left-0 w-full h-4 flex items-center justify-center">
              <span className="text-xs font-semibold text-white drop-shadow-lg">
                {progress ? `${progress.toFixed(1)}%` : '0%'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Statistics Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Cards Scraped */}
        <div className="bg-gradient-to-br from-blue-50 to-blue-100 rounded-xl p-6 border border-blue-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-blue-600 mb-1">Cards Scraped</p>
              <p className="text-3xl font-bold text-blue-800">{formatNumber(cardsScraped)}</p>
            </div>
            <div className="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zM21 5a2 2 0 00-2-2h-4a2 2 0 00-2 2v12a4 4 0 004 4h4a4 4 0 004-4V5z" />
              </svg>
            </div>
          </div>
        </div>

        {/* Scraping Speed */}
        <div className="bg-gradient-to-br from-green-50 to-green-100 rounded-xl p-6 border border-green-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-green-600 mb-1">Speed</p>
              <p className="text-3xl font-bold text-green-800">{Math.round(cardsPerMinute || 0)}</p>
              <p className="text-xs text-green-600">cards/min</p>
            </div>
            <div className="w-12 h-12 bg-green-500 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
          </div>
        </div>

        {/* Time Remaining */}
        <div className="bg-gradient-to-br from-purple-50 to-purple-100 rounded-xl p-6 border border-purple-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-purple-600 mb-1">Time Remaining</p>
              <p className="text-2xl font-bold text-purple-800">{formatDuration(estimatedTimeRemaining)}</p>
            </div>
            <div className="w-12 h-12 bg-purple-500 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
        </div>

        {/* Elapsed Time */}
        <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-xl p-6 border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600 mb-1">Elapsed Time</p>
              <p className="text-2xl font-bold text-gray-800">{formatElapsedTime(startTime)}</p>
            </div>
            <div className="w-12 h-12 bg-gray-500 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {/* Additional Info Panel */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <h4 className="text-lg font-semibold text-gray-800 mb-4">Scraping Details</h4>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="text-center">
            <p className="text-sm text-gray-500 mb-1">Current Page</p>
            <p className="text-2xl font-bold text-gray-800">{formatNumber(currentPage || 0)}</p>
          </div>
          <div className="text-center">
            <p className="text-sm text-gray-500 mb-1">Total Pages</p>
            <p className="text-2xl font-bold text-gray-800">{formatNumber(totalPages || 0)}</p>
          </div>
          <div className="text-center">
            <p className="text-sm text-gray-500 mb-1">Progress</p>
            <p className="text-2xl font-bold text-gray-800">{progress ? `${progress.toFixed(1)}%` : '0%'}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProgressBar; 