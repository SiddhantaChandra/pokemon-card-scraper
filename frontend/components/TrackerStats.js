import { useState } from 'react';
import { useDarkMode } from '../lib/darkModeContext';

export default function TrackerStats({ stats, workerStatus }) {
  const { isDarkMode } = useDarkMode();
  const [isOpen, setIsOpen] = useState(false);

  const formatDate = (dateString) => {
    if (!dateString) return 'Never';
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const outOfStockCount = (stats?.total_trackers || 0) - (stats?.in_stock_trackers || 0);

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`w-full flex items-center justify-between px-4 py-3 rounded-lg border ${
          isDarkMode 
            ? 'bg-gray-800 border-gray-700 text-white hover:bg-gray-700' 
            : 'bg-white border-gray-200 text-gray-900 hover:bg-gray-50'
        } transition-colors`}
      >
        <div className="flex items-center space-x-3">
          <div className={`w-3 h-3 rounded-full ${
            workerStatus?.is_running ? 'bg-green-500' : 'bg-red-500'
          }`}></div>
          <span className="font-medium">
            Tracker Stats ({stats?.total_trackers || 0} total)
          </span>
        </div>
        <svg
          className={`w-5 h-5 transition-transform ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div className={`absolute top-full left-0 right-0 mt-1 rounded-lg border shadow-lg z-10 ${
          isDarkMode 
            ? 'bg-gray-800 border-gray-700' 
            : 'bg-white border-gray-200'
        }`}>
          <div className="p-4 space-y-3">
            <div className="flex justify-between">
              <span className={isDarkMode ? 'text-gray-300' : 'text-gray-600'}>In Stock:</span>
              <span className={`font-medium ${isDarkMode ? 'text-green-400' : 'text-green-600'}`}>
                {stats?.in_stock_trackers || 0}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span className={isDarkMode ? 'text-gray-300' : 'text-gray-600'}>Out of Stock:</span>
              <span className={`font-medium ${isDarkMode ? 'text-red-400' : 'text-red-600'}`}>
                {outOfStockCount}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span className={isDarkMode ? 'text-gray-300' : 'text-gray-600'}>Last Scan:</span>
              <span className={`font-medium ${isDarkMode ? 'text-gray-200' : 'text-gray-900'}`}>
                {workerStatus?.last_scan_time ? formatDate(workerStatus.last_scan_time) : 'Never'}
              </span>
            </div>
            
            <div className="flex justify-between">
              <span className={isDarkMode ? 'text-gray-300' : 'text-gray-600'}>Status:</span>
              <span className={`font-medium ${
                workerStatus?.is_running 
                  ? isDarkMode ? 'text-green-400' : 'text-green-600'
                  : isDarkMode ? 'text-red-400' : 'text-red-600'
              }`}>
                {workerStatus?.is_running ? 'Running' : 'Stopped'}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
} 