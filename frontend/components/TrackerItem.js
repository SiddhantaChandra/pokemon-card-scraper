import { useState } from 'react';
import Image from 'next/image';
import { useDarkMode } from '../lib/darkModeContext';

export default function TrackerItem({ tracker, onDelete }) {
  const { isDarkMode } = useDarkMode();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [imageLoading, setImageLoading] = useState(true);

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(tracker.id);
    } catch (error) {
      console.error('Failed to delete tracker:', error);
    } finally {
      setIsDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  const formatPrice = (price) => {
    if (!price || price === 0) return 'N/A';
    return `¥${price.toLocaleString()}`;
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'Never';
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString();
  };

  const handleImageLoad = () => {
    setImageLoading(false);
  };

  const handleImageError = () => {
    setImageError(true);
    setImageLoading(false);
  };

  const handleCardClick = () => {
    if (tracker.url) {
      window.open(tracker.url, '_blank', 'noopener,noreferrer');
    }
  };

  // Get border color and stock tab styling based on stock status
  const getStockStyling = () => {
    if (tracker.in_stock) {
      return {
        borderColor: 'border-green-500 dark:border-green-400',
        tabColor: 'bg-green-500 text-white',
        statusText: 'IN STOCK'
      };
    } else {
      return {
        borderColor: 'border-red-500 dark:border-red-400',
        tabColor: 'bg-red-500 text-white',
        statusText: 'OUT OF STOCK'
      };
    }
  };

  const stockStyling = getStockStyling();

  return (
    <>
      <div className={`group bg-white dark:bg-gray-800 rounded-lg shadow-md hover:shadow-lg transition-all duration-200 overflow-hidden border-2 ${stockStyling.borderColor} hover:border-opacity-80`}>
        {/* Image Container */}
        <div className="relative aspect-[3/4] bg-gray-100 dark:bg-gray-700 overflow-hidden">
          {imageLoading && tracker.image_url && (
            <div className="absolute inset-0 flex items-center justify-center z-10">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400"></div>
            </div>
          )}
          
          {tracker.image_url && !imageError ? (
            <Image
              src={tracker.image_url}
              alt={tracker.name || 'Pokemon Card'}
              fill
              className={`object-cover transition-transform duration-200 group-hover:scale-105${
                imageLoading ? 'opacity-0' : 'opacity-100'
              }`}
              onLoad={handleImageLoad}
              onError={handleImageError}
              loading="lazy"
              unoptimized={true}
              crossOrigin="anonymous"
              onClick={handleCardClick}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-gray-200 dark:bg-gray-600">
              <div className="text-center text-gray-500 dark:text-gray-400">
                <svg className="mx-auto h-12 w-12 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <p className="text-xs">No Image</p>
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div className="absolute bottom-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200 z-20">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowDeleteConfirm(true);
              }}
              className="bg-black bg-opacity-75 text-white p-2 rounded hover:bg-red-600 transition-colors"
              title="Delete tracker"
            >
              <svg className="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        {/* Card Content */}
        <div className="px-4 py-2">
          {/* Card Name */}
          <h3 className="font-semibold text-gray-900 dark:text-white text-sm mb-2">
            {tracker.name}
          </h3>

          {/* Stock Status Tab - Prominent */}
          <div className="m-0">
            <span className={`cursor-pointer inline-block w-full text-center py-2 px-3 rounded-lg text-sm font-bold ${stockStyling.tabColor}`} onClick={handleCardClick}>
              {stockStyling.statusText}
            </span>
          </div>

          {/* Price and Last Checked */}
          <div className="space-y-2 my-1">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Price:</span>
              <span className="text-lg font-bold text-gray-900 dark:text-white">
                {formatPrice(tracker.price)}
              </span>
            </div>

          </div>

          {/* View Button */}
          <button
            onClick={handleCardClick}
            className="w-full bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white py-2 px-3 rounded text-sm font-medium transition-colors duration-200 flex items-center justify-center space-x-1"
          >
            <span>View</span>
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
          </button>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className={`rounded-lg p-6 max-w-sm w-full ${
            isDarkMode ? 'bg-gray-800' : 'bg-white'
          }`}>
            <h3 className={`text-lg font-semibold mb-2 ${
              isDarkMode ? 'text-white' : 'text-gray-900'
            }`}>
              Delete Tracker
            </h3>
            <p className={`text-sm mb-4 ${
              isDarkMode ? 'text-gray-300' : 'text-gray-600'
            }`}>
              Are you sure you want to delete "{tracker.name}"? This action cannot be undone.
            </p>
            
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                disabled={isDeleting}
                className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors disabled:opacity-50 ${
                  isDarkMode
                    ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                    : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={isDeleting}
                className="px-4 py-2 text-sm font-medium bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
} 