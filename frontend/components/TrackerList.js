import { useState } from 'react';
import { useDarkMode } from '../lib/darkModeContext';
import TrackerItem from './TrackerItem';

export default function TrackerList({ trackers, onDelete, loading }) {
  const { isDarkMode } = useDarkMode();
  const [filter, setFilter] = useState('all'); // all, in_stock, out_of_stock
  const [sortBy, setSortBy] = useState('created_at'); // created_at, name, price, last_checked
  const [sortOrder, setSortOrder] = useState('desc'); // asc, desc

  // Filter trackers based on selected filter
  const filteredTrackers = trackers.filter(tracker => {
    if (filter === 'in_stock') return tracker.in_stock;
    if (filter === 'out_of_stock') return !tracker.in_stock;
    return true; // 'all'
  });

  // Sort trackers
  const sortedTrackers = [...filteredTrackers].sort((a, b) => {
    let aValue, bValue;
    
    switch (sortBy) {
      case 'name':
        aValue = a.name.toLowerCase();
        bValue = b.name.toLowerCase();
        break;
      case 'price':
        aValue = a.price || 0;
        bValue = b.price || 0;
        break;
      case 'last_checked':
        aValue = new Date(a.last_checked || 0);
        bValue = new Date(b.last_checked || 0);
        break;
      case 'created_at':
      default:
        aValue = new Date(a.created_at);
        bValue = new Date(b.created_at);
        break;
    }
    
    if (sortOrder === 'asc') {
      return aValue > bValue ? 1 : -1;
    } else {
      return aValue < bValue ? 1 : -1;
    }
  });

  const getFilterCount = (filterType) => {
    if (filterType === 'in_stock') return trackers.filter(t => t.in_stock).length;
    if (filterType === 'out_of_stock') return trackers.filter(t => !t.in_stock).length;
    return trackers.length;
  };

  return (
    <div>
      {/* Header with filters and sort */}
      <div className={`rounded-lg shadow-md p-4 mb-6 border ${
        isDarkMode ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'
      }`}>
        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <h2 className={`text-xl font-semibold ${
            isDarkMode ? 'text-white' : 'text-gray-900'
          }`}>
            Your Trackers ({sortedTrackers.length})
          </h2>
          
          <div className="flex flex-col sm:flex-row gap-4">
            {/* Filter Buttons */}
            <div className="flex gap-2">
              <button
                onClick={() => setFilter('all')}
                className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                  filter === 'all'
                    ? isDarkMode
                      ? 'bg-blue-600 text-white'
                      : 'bg-blue-500 text-white'
                    : isDarkMode
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                All ({getFilterCount('all')})
              </button>
              <button
                onClick={() => setFilter('in_stock')}
                className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                  filter === 'in_stock'
                    ? 'bg-green-600 text-white'
                    : isDarkMode
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                In Stock ({getFilterCount('in_stock')})
              </button>
              <button
                onClick={() => setFilter('out_of_stock')}
                className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                  filter === 'out_of_stock'
                    ? 'bg-red-600 text-white'
                    : isDarkMode
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                Out of Stock ({getFilterCount('out_of_stock')})
              </button>
            </div>

            {/* Sort Controls */}
            <div className="flex gap-2">
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value)}
                className={`px-3 py-1 rounded border text-sm ${
                  isDarkMode
                    ? 'bg-gray-700 border-gray-600 text-white'
                    : 'bg-white border-gray-300 text-gray-900'
                }`}
              >
                <option value="created_at">Date Added</option>
                <option value="name">Name</option>
                <option value="price">Price</option>
                <option value="last_checked">Last Checked</option>
              </select>
              
              <button
                onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
                className={`px-2 py-1 rounded border text-sm ${
                  isDarkMode
                    ? 'bg-gray-700 border-gray-600 text-white hover:bg-gray-600'
                    : 'bg-white border-gray-300 text-gray-900 hover:bg-gray-50'
                }`}
                title={`Sort ${sortOrder === 'asc' ? 'descending' : 'ascending'}`}
              >
                {sortOrder === 'asc' ? '↑' : '↓'}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Loading State */}
      {loading && (
        <div className={`text-center py-8 ${
          isDarkMode ? 'text-gray-400' : 'text-gray-500'
        }`}>
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-current"></div>
          <p className="mt-2">Loading trackers...</p>
        </div>
      )}

      {/* Trackers Grid - Similar to CardGrid */}
      {!loading && sortedTrackers.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {sortedTrackers.map((tracker) => (
            <TrackerItem
              key={tracker.id}
              tracker={tracker}
              onDelete={onDelete}
            />
          ))}
        </div>
      )}

      {/* Empty State */}
      {!loading && sortedTrackers.length === 0 && filteredTrackers.length !== trackers.length && (
        <div className={`text-center py-8 ${
          isDarkMode ? 'text-gray-400' : 'text-gray-500'
        }`}>
          <p>No trackers match the current filter.</p>
          <button
            onClick={() => setFilter('all')}
            className={`mt-2 text-sm underline ${
              isDarkMode ? 'text-blue-400 hover:text-blue-300' : 'text-blue-600 hover:text-blue-700'
            }`}
          >
            Show all trackers
          </button>
        </div>
      )}
    </div>
  );
} 