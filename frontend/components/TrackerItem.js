import React, { useState } from 'react';
import { apiHelpers } from '../lib/api';

const TrackerItem = ({ tracker, onUpdate, onDelete }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState(tracker.name);
  const [isUpdating, setIsUpdating] = useState(false);

  const handleSaveEdit = async () => {
    if (editName.trim() === tracker.name) {
      setIsEditing(false);
      return;
    }

    try {
      setIsUpdating(true);
      await onUpdate(tracker.id, { name: editName.trim() });
      setIsEditing(false);
    } catch (error) {
      console.error('Failed to update tracker:', error);
      alert('Failed to update tracker name');
    } finally {
      setIsUpdating(false);
    }
  };

  const handleCancelEdit = () => {
    setEditName(tracker.name);
    setIsEditing(false);
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter') {
      handleSaveEdit();
    } else if (e.key === 'Escape') {
      handleCancelEdit();
    }
  };

  const getStockStatus = () => {
    if (tracker.in_stock) {
      return {
        text: 'In Stock',
        color: 'text-green-600 dark:text-green-400',
        bgColor: 'bg-green-100 dark:bg-green-900',
        icon: '✅',
      };
    } else {
      return {
        text: 'Out of Stock',
        color: 'text-red-600 dark:text-red-400',
        bgColor: 'bg-red-100 dark:bg-red-900',
        icon: '❌',
      };
    }
  };

  const status = getStockStatus();
  const lastUpdated = tracker.last_updated 
    ? apiHelpers.formatDate(tracker.last_updated)
    : 'Never';

  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6 shadow-sm hover:shadow-md transition-shadow">
      <div className="flex flex-col lg:flex-row gap-4">
        {/* Image */}
        <div className="lg:w-32 lg:h-32 w-full h-48 bg-gray-100 dark:bg-gray-700 rounded-lg overflow-hidden flex-shrink-0">
          {tracker.image ? (
            <img
              src={tracker.image}
              alt={tracker.name || 'Product'}
              className="w-full h-full object-cover"
              onError={(e) => {
                e.target.src = '/images/card-placeholder.svg';
              }}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-gray-400 dark:text-gray-500">
              <span className="text-2xl">🎴</span>
            </div>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-2 mb-3">
            <div className="flex-1 min-w-0">
              {isEditing ? (
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={handleKeyPress}
                    className="flex-1 px-2 py-1 text-lg font-semibold border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    autoFocus
                    disabled={isUpdating}
                  />
                  <button
                    onClick={handleSaveEdit}
                    disabled={isUpdating}
                    className="px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                  >
                    ✓
                  </button>
                  <button
                    onClick={handleCancelEdit}
                    disabled={isUpdating}
                    className="px-3 py-1 bg-gray-600 text-white rounded hover:bg-gray-700 disabled:opacity-50"
                  >
                    ✕
                  </button>
                </div>
              ) : (
                <h3 
                  className="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate cursor-pointer hover:text-blue-600 dark:hover:text-blue-400"
                  onClick={() => setIsEditing(true)}
                  title={tracker.name || 'Unnamed Tracker'}
                >
                  {tracker.name || 'Unnamed Tracker'}
                </h3>
              )}
            </div>

            {/* Stock Status */}
            <div className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-sm font-medium ${status.bgColor} ${status.color}`}>
              <span>{status.icon}</span>
              {status.text}
            </div>
          </div>

          {/* URL */}
          <div className="mb-3">
            <a
              href={tracker.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 dark:text-blue-400 hover:underline text-sm break-all"
            >
              {tracker.url}
            </a>
          </div>

          {/* Details */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-gray-500 dark:text-gray-400">Price:</span>
              <div className="font-medium text-gray-900 dark:text-gray-100">
                {tracker.price_yen || 'Unknown'}
              </div>
            </div>
            
            <div>
              <span className="text-gray-500 dark:text-gray-400">Last Updated:</span>
              <div className="font-medium text-gray-900 dark:text-gray-100">
                {lastUpdated}
              </div>
            </div>

            <div>
              <span className="text-gray-500 dark:text-gray-400">Added:</span>
              <div className="font-medium text-gray-900 dark:text-gray-100">
                {apiHelpers.formatDate(tracker.created_at)}
              </div>
            </div>

            <div>
              <span className="text-gray-500 dark:text-gray-400">ID:</span>
              <div className="font-mono text-xs text-gray-600 dark:text-gray-400">
                {tracker.id}
              </div>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex lg:flex-col gap-2 lg:w-auto w-full">
          <button
            onClick={() => window.open(tracker.url, '_blank')}
            className="flex-1 lg:flex-none px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors text-sm"
          >
            View Product
          </button>
          
          <button
            onClick={() => setIsEditing(!isEditing)}
            className="flex-1 lg:flex-none px-3 py-2 bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors text-sm"
          >
            Edit Name
          </button>
          
          <button
            onClick={() => onDelete(tracker.id)}
            className="flex-1 lg:flex-none px-3 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors text-sm"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
};

export default TrackerItem; 