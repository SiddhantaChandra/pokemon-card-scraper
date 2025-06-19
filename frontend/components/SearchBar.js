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

  // Available sort options
  const availableSortOptions = [
    { value: 'date_desc', label: 'Newest First' },
    { value: 'date_asc', label: 'Oldest First' },
    { value: 'price_asc', label: 'Price: Low to High' },
    { value: 'price_desc', label: 'Price: High to Low' },
    { value: 'name_asc', label: 'Name: A to Z' },
    { value: 'name_desc', label: 'Name: Z to A' },
    { value: 'stock_desc', label: 'Stock: High to Low' }
  ];

  // Available condition options
  const conditionOptions = [
    { value: 'Perfect', label: 'Perfect', color: 'bg-green-100 text-green-800' },
    { value: 'A+', label: 'A+', color: 'bg-blue-100 text-blue-800' },
    { value: 'A', label: 'A', color: 'bg-blue-100 text-blue-800' },
    { value: 'A-', label: 'A-', color: 'bg-blue-100 text-blue-800' },
    { value: 'B+', label: 'B+', color: 'bg-yellow-100 text-yellow-800' },
    { value: 'B', label: 'B', color: 'bg-yellow-100 text-yellow-800' },
    { value: 'B-', label: 'B-', color: 'bg-yellow-100 text-yellow-800' },
    { value: 'C', label: 'C', color: 'bg-red-100 text-red-800' },
    { value: 'D', label: 'D', color: 'bg-red-100 text-red-800' }
  ];

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
    onFiltersChange({ ...filters, [key]: value, page: 1 });
  };

  // Handle price range change
  const handlePriceRangeChange = (key, value) => {
    setTempPriceRange(prev => ({ ...prev, [key]: value }));
  };

  // Apply price range filter
  const applyPriceRange = () => {
    const newFilters = { ...filters, page: 1 };
    
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

  // Handle condition filter changes
  const handleConditionChange = (condition) => {
    const currentConditions = filters.conditions || [];
    let newConditions;
    
    if (currentConditions.includes(condition)) {
      newConditions = currentConditions.filter(c => c !== condition);
    } else {
      newConditions = [...currentConditions, condition];
    }
    
    handleFilterChange('conditions', newConditions);
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

  const hasActiveFilters = query || filters.min_price || filters.max_price || filters.in_stock || 
    filters.set || filters.rarity || (filters.conditions && filters.conditions.length > 0) || 
    filters.sort !== 'date_desc';

  return (
    <div className={`bg-white rounded-lg shadow-md border border-gray-200 ${className}`}>
      {/* Main Search Bar */}
      <div className="p-6">
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
              className="block w-full pl-10 pr-12 py-3 border border-gray-300 rounded-lg leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-lg"
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
            <div className="absolute z-10 mt-1 w-full bg-white shadow-lg rounded-lg border border-gray-200 max-h-60 overflow-auto">
              {suggestions.map((suggestion, index) => (
                <button
                  key={suggestion}
                  onClick={() => handleSuggestionClick(suggestion)}
                  className={`w-full text-left px-4 py-3 hover:bg-gray-50 ${
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
              className={`flex items-center space-x-2 px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                showFilters
                  ? 'bg-blue-100 text-blue-700 border border-blue-200'
                  : 'bg-gray-100 text-gray-700 border border-gray-200 hover:bg-gray-200'
              }`}
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
              </svg>
              <span>Filters</span>
              {hasActiveFilters && (
                <span className="ml-1 inline-flex items-center justify-center w-5 h-5 text-xs font-medium text-white bg-blue-600 rounded-full">
                  !
                </span>
              )}
            </button>

            {/* Sort Dropdown */}
            <div className="relative">
              <select
                value={filters.sort || 'date_desc'}
                onChange={(e) => handleFilterChange('sort', e.target.value)}
                className="appearance-none bg-white border border-gray-200 rounded-lg px-4 py-2 pr-8 text-sm font-medium text-gray-700 hover:border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                {availableSortOptions.map(option => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              <div className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>
          </div>

          {hasActiveFilters && (
            <button
              onClick={clearFilters}
              className="text-sm text-gray-600 hover:text-gray-800 font-medium"
            >
              Clear All Filters
            </button>
          )}
        </div>
      </div>

      {/* Advanced Filters */}
      {showFilters && (
        <div className="border-t border-gray-200 bg-gray-50">
          <div className="p-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
              
              {/* Price Range */}
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">Price Range (¥)</label>
                <div className="flex space-x-2">
                  <input
                    type="number"
                    placeholder="Min"
                    value={tempPriceRange.min}
                    onChange={(e) => handlePriceRangeChange('min', e.target.value)}
                    onBlur={applyPriceRange}
                    onKeyPress={(e) => e.key === 'Enter' && applyPriceRange()}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                  />
                  <span className="self-center text-gray-500">-</span>
                  <input
                    type="number"
                    placeholder="Max"
                    value={tempPriceRange.max}
                    onChange={(e) => handlePriceRangeChange('max', e.target.value)}
                    onBlur={applyPriceRange}
                    onKeyPress={(e) => e.key === 'Enter' && applyPriceRange()}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Set Name */}
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">Set Name</label>
                <input
                  type="text"
                  placeholder="e.g., Base Set, Jungle..."
                  value={filters.set || ''}
                  onChange={(e) => handleFilterChange('set', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              {/* Rarity */}
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">Rarity</label>
                <input
                  type="text"
                  placeholder="e.g., Rare, Common, Uncommon..."
                  value={filters.rarity || ''}
                  onChange={(e) => handleFilterChange('rarity', e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>

              {/* Stock Filter */}
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">Availability</label>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={filters.in_stock || false}
                    onChange={(e) => handleFilterChange('in_stock', e.target.checked)}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <span className="ml-2 text-sm text-gray-700">In stock only</span>
                </label>
              </div>
            </div>

            {/* Card Conditions */}
            <div className="mt-6">
              <label className="block text-sm font-medium text-gray-700 mb-3">Card Condition</label>
              <div className="flex flex-wrap gap-2">
                {conditionOptions.map(condition => (
                  <button
                    key={condition.value}
                    onClick={() => handleConditionChange(condition.value)}
                    className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium transition-all ${
                      (filters.conditions || []).includes(condition.value)
                        ? `${condition.color} ring-2 ring-offset-1 ring-blue-500`
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                    }`}
                  >
                    {condition.label}
                  </button>
                ))}
              </div>
              <p className="text-xs text-gray-500 mt-2">
                Click to select multiple conditions. Perfect = No condition indicator in title.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
} 