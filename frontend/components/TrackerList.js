import React, { useState } from 'react';
import { useTrackers } from '../hooks/useTrackers';
import TrackerItem from './TrackerItem';
import AddTrackerForm from './AddTrackerForm';
import TrackerFilters from './TrackerFilters';
import Pagination from './Pagination';

const TrackerList = () => {
  const [showAddForm, setShowAddForm] = useState(false);
  const {
    trackers,
    pagination,
    filters,
    isLoading,
    error,
    addTracker,
    updateTracker,
    deleteTracker,
    updateFilters,
    refresh,
  } = useTrackers();

  const handleAddTracker = async (url, name) => {
    try {
      await addTracker(url, name);
      setShowAddForm(false);
    } catch (error) {
      console.error('Failed to add tracker:', error);
      throw error; // Re-throw to let form handle error display
    }
  };

  const handleUpdateTracker = async (id, data) => {
    try {
      await updateTracker(id, data);
    } catch (error) {
      console.error('Failed to update tracker:', error);
      throw error;
    }
  };

  const handleDeleteTracker = async (id) => {
    if (window.confirm('Are you sure you want to delete this tracker?')) {
      try {
        await deleteTracker(id);
      } catch (error) {
        console.error('Failed to delete tracker:', error);
        alert('Failed to delete tracker. Please try again.');
      }
    }
  };

  const handlePageChange = (page) => {
    updateFilters({ page });
  };

  const handleFiltersChange = (newFilters) => {
    updateFilters(newFilters);
  };

  if (error) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900 border border-red-200 dark:border-red-700 rounded-lg">
        <p className="text-red-800 dark:text-red-200">
          Failed to load trackers: {error.message}
        </p>
        <button
          onClick={refresh}
          className="mt-2 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Stock Trackers
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            Monitor Pokemon card stock changes
          </p>
        </div>
        
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          <span>+</span>
          Add Tracker
        </button>
      </div>

      {/* Add Tracker Form */}
      {showAddForm && (
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Add New Tracker
          </h3>
          <AddTrackerForm
            onSubmit={handleAddTracker}
            onCancel={() => setShowAddForm(false)}
          />
        </div>
      )}

      {/* Filters */}
      <TrackerFilters
        filters={filters}
        onChange={handleFiltersChange}
        isLoading={isLoading}
      />

      {/* Tracker List */}
      {isLoading && trackers.length === 0 ? (
        <div className="flex justify-center items-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      ) : trackers.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-gray-400 dark:text-gray-500 text-6xl mb-4">📊</div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
            No trackers yet
          </h3>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            Add your first tracker to start monitoring stock changes
          </p>
          <button
            onClick={() => setShowAddForm(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            Add Your First Tracker
          </button>
        </div>
      ) : (
        <>
          <div className="grid gap-4">
            {trackers.map((tracker) => (
              <TrackerItem
                key={tracker.id}
                tracker={tracker}
                onUpdate={handleUpdateTracker}
                onDelete={handleDeleteTracker}
              />
            ))}
          </div>

          {/* Pagination */}
          {pagination.total_pages > 1 && (
            <Pagination
              currentPage={pagination.page}
              totalPages={pagination.total_pages}
              totalItems={pagination.total}
              itemsPerPage={pagination.page_size}
              onPageChange={handlePageChange}
            />
          )}
        </>
      )}

      {/* Loading overlay */}
      {isLoading && trackers.length > 0 && (
        <div className="fixed inset-0 bg-black bg-opacity-20 flex justify-center items-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-4 flex items-center gap-3">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
            <span className="text-gray-900 dark:text-gray-100">Updating...</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default TrackerList; 