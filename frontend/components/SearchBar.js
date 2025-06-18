import { useState, useRef, useEffect } from 'react';
import { useSearchSuggestions } from '../hooks/useCards';
import { useSearchSuggestions as useSuggestionState } from '../hooks/useSearch';

export default function SearchBar({
  query,
  onQueryChange,
  filters,
  onFiltersChange,
  sortOptions = [],
  className = ''
}) {
  const [showFilters, setShowFilters] = useState(false);
  const [tempPriceRange, setTempPriceRange] = useState({
    min: filters.min_price || '',
    max: filters.max_price || ''
  });
  
  const searchInputRef = useRef(null);
  const suggestionsRef = useRef(null);
  
  // Search suggestions
  const { suggestions, isLoading: suggestionsLoading } = useSearchSuggestions(query, 8);
  const {
    isOpen: suggestionsOpen,
    selectedIndex,
    openSuggestions,
    closeSuggestions,
    selectNext,
    selectPrev,
    selectSuggestion
  } = useSuggestionState();

  // Handle search input changes
  const handleQueryChange = (e) => {
    const value = e.target.value;
    onQueryChange(value);
    
    if (value.length > 2) {
      openSuggestions();
    } else {
      closeSuggestions();
    }
  };

  // Handle keyboard navigation in suggestions
  const handleKeyDown = (e) => {
    if (!suggestionsOpen || suggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        selectNext(suggestions.length);
        break;
      case 'ArrowUp':
        e.preventDefault();
        selectPrev(suggestions.length);
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < suggestions.length) {
          onQueryChange(suggestions[selectedIndex]);
          closeSuggestions();
          searchInputRef.current?.blur();
        }
        break;
      case 'Escape':
        closeSuggestions();
        searchInputRef.current?.blur();
        break;
    }
  };

  // Handle suggestion click
  const handleSuggestionClick = (suggestion) => {
    onQueryChange(suggestion);
    closeSuggestions();
    searchInputRef.current?.focus();
  };

  // Handle filter changes
  const handleFilterChange = (key, value) => {
    onFiltersChange({ ...filters, [key]: value });
  };

  // Handle price range change
  const handlePriceRangeChange = (key, value) => {
    setTempPriceRange(prev => ({ ...prev, [key]: value }));
  };

  // Apply price range filter
  const applyPriceRange = () => {
    const newFilters = { ...filters };
    
    if (tempPriceRange.min) {
      newFilters.min_price = parseFloat(tempPriceRange.min);
    } else {
      delete newFilters.min_price;
    }
    
    if (tempPriceRange.max) {
      newFilters.max_price = parseFloat(tempPriceRange.max);
    } else {
      delete newFilters.max_price;
    }
    
    onFiltersChange(newFilters);
  };

  // Clear all filters
  const clearFilters = () => {
    onQueryChange('');
    onFiltersChange({
      page: 1,
      page_size: 20,
      sort: 'date_desc'
    });
    setTempPriceRange({ min: '', max: '' });
    closeSuggestions();
  };

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (suggestionsRef.current && !suggestionsRef.current.contains(event.target)) {
        closeSuggestions();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [closeSuggestions]);

  // Update temp price range when filters change externally
  useEffect(() => {
    setTempPriceRange({
      min: filters.min_price || '',
      max: filters.max_price || ''
    });
  }, [filters.min_price, filters.max_price]);

  const hasActiveFilters = query || filters.min_price || filters.max_price || filters.in_stock || filters.sort !== 'date_desc';

  return (
    <div className={`bg-white rounded-lg shadow-md border border-gray-200 ${className}`}>
      {/* Main Search Bar */}
      <div className="p-4">
        <div className="relative" ref={suggestionsRef}>
          {/* Search Input */}
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            <input
              ref={searchInputRef}
              type="text"
              placeholder="Search Pokemon cards..."
              value={query}
              onChange={handleQueryChange}
              onKeyDown={handleKeyDown}
              onFocus={() => query.length > 2 && openSuggestions()}
              className="block w-full pl-10 pr-12 py-3 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            />
            {query && (
              <button
                onClick={() => onQueryChange('')}
                className="absolute inset-y-0 right-0 pr-3 flex items-center"
              >
                <svg className="h-5 w-5 text-gray-400 hover:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>

          {/* Search Suggestions */}
          {suggestionsOpen && suggestions.length > 0 && (
            <div className="absolute z-10 mt-1 w-full bg-white shadow-lg rounded-md border border-gray-200 max-h-60 overflow-auto">
              {suggestions.map((suggestion, index) => (
                <button
                  key={suggestion}
                  onClick={() => handleSuggestionClick(suggestion)}
                  className={`w-full text-left px-4 py-2 hover:bg-gray-50 ${
                    index === selectedIndex ? 'bg-blue-50 text-blue-600' : 'text-gray-900'
                  }`}
                >
                  {suggestion}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Filter Toggle and Actions */}
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center space-x-4">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className="flex items-center space-x-2 text-sm text-gray-600 hover:text-gray-900"
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
              </svg>
              <span>Filters</span>
              {showFilters ? (
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                </svg>
              ) : (
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              )}
            </button>

            {hasActiveFilters && (
              <button
                onClick={clearFilters}
                className="text-sm text-red-600 hover:text-red-800 flex items-center space-x-1"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
                <span>Clear All</span>
              </button>
            )}
          </div>

          {/* Sort Dropdown */}
          <div className="flex items-center space-x-2">
            <label className="text-sm text-gray-600">Sort:</label>
            <select
              value={filters.sort || 'date_desc'}
              onChange={(e) => handleFilterChange('sort', e.target.value)}
              className="text-sm border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="date_desc">Newest First</option>
              <option value="date_asc">Oldest First</option>
              <option value="price_asc">Price: Low to High</option>
              <option value="price_desc">Price: High to Low</option>
              <option value="name_asc">Name: A to Z</option>
              <option value="name_desc">Name: Z to A</option>
              <option value="stock_desc">Stock: High to Low</option>
            </select>
          </div>
        </div>
      </div>

      {/* Advanced Filters */}
      {showFilters && (
        <div className="border-t border-gray-200 p-4 bg-gray-50">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Price Range */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Price Range (¥)</label>
              <div className="flex items-center space-x-2">
                <input
                  type="number"
                  placeholder="Min"
                  value={tempPriceRange.min}
                  onChange={(e) => handlePriceRangeChange('min', e.target.value)}
                  onBlur={applyPriceRange}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                />
                <span className="text-gray-500">-</span>
                <input
                  type="number"
                  placeholder="Max"
                  value={tempPriceRange.max}
                  onChange={(e) => handlePriceRangeChange('max', e.target.value)}
                  onBlur={applyPriceRange}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
            </div>

            {/* Stock Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Availability</label>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={filters.in_stock || false}
                  onChange={(e) => handleFilterChange('in_stock', e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">In Stock Only</span>
              </label>
            </div>

            {/* Page Size */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Items per Page</label>
              <select
                value={filters.page_size || 20}
                onChange={(e) => handleFilterChange('page_size', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </div>
          </div>
        </div>
      )}
    </div>
  );
} 