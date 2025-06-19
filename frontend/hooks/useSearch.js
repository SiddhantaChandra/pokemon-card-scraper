import { useState, useCallback, useEffect } from 'react';
import { useDebounce } from './useDebounce';

// Custom hook for managing search state
export function useSearch(initialQuery = '', initialFilters = {}) {
  const [query, setQuery] = useState(initialQuery);
  const [filters, setFilters] = useState({
    page: 1,
    page_size: 20,
    in_stock: false,
    sort: 'date_desc',
    ...initialFilters,
  });
  
  // Debounced query for API calls
  const debouncedQuery = useDebounce(query, 500);

  // Update a single filter
  const updateFilter = useCallback((key, value) => {
    setFilters(prev => ({
      ...prev,
      [key]: value,
      // Reset to page 1 when filters change (except for page changes)
      ...(key !== 'page' && { page: 1 }),
    }));
  }, []);

  // Update multiple filters at once
  const updateFilters = useCallback((newFilters) => {
    setFilters(prev => ({
      ...prev,
      ...newFilters,
      // Reset to page 1 when filters change
      page: 1,
    }));
  }, []);

  // Set search query
  const setSearchQuery = useCallback((newQuery) => {
    setQuery(newQuery);
    // Reset to page 1 when query changes
    setFilters(prev => ({ ...prev, page: 1 }));
  }, []);

  // Clear all filters and query
  const clearSearch = useCallback(() => {
    setQuery('');
    setFilters({
      page: 1,
      page_size: 20,
      in_stock: false,
      sort: 'date_desc',
    });
  }, []);

  // Go to specific page
  const goToPage = useCallback((page) => {
    updateFilter('page', page);
  }, [updateFilter]);

  // Go to next page
  const nextPage = useCallback(() => {
    setFilters(prev => ({ ...prev, page: prev.page + 1 }));
  }, []);

  // Go to previous page
  const prevPage = useCallback(() => {
    setFilters(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }));
  }, []);

  // Set price range
  const setPriceRange = useCallback((minPrice, maxPrice) => {
    updateFilters({
      min_price: minPrice || undefined,
      max_price: maxPrice || undefined,
    });
  }, [updateFilters]);

  // Toggle in-stock filter
  const toggleInStockFilter = useCallback(() => {
    updateFilter('in_stock', !filters.in_stock);
  }, [filters.in_stock, updateFilter]);

  // Set sort option
  const setSortOption = useCallback((sortOption) => {
    updateFilter('sort', sortOption);
  }, [updateFilter]);

  // Get URL search params (for sharing/bookmarking)
  const getSearchParams = useCallback(() => {
    const params = new URLSearchParams();
    
    if (query) params.set('q', query);
    if (filters.page > 1) params.set('page', filters.page.toString());
    if (filters.page_size !== 20) params.set('page_size', filters.page_size.toString());
    if (filters.min_price) params.set('min_price', filters.min_price.toString());
    if (filters.max_price) params.set('max_price', filters.max_price.toString());
    if (filters.in_stock) params.set('in_stock', 'true');
    if (filters.set) params.set('set', filters.set);
    if (filters.rarity) params.set('rarity', filters.rarity);
    if (filters.conditions && filters.conditions.length > 0) {
      params.set('conditions', filters.conditions.join(','));
    }
    if (filters.sort !== 'date_desc') params.set('sort', filters.sort);
    
    return params.toString();
  }, [query, filters]);

  // Load search from URL params
  const loadFromSearchParams = useCallback((searchParams) => {
    const params = new URLSearchParams(searchParams);
    
    setQuery(params.get('q') || '');
    setFilters({
      page: parseInt(params.get('page')) || 1,
      page_size: parseInt(params.get('page_size')) || 20,
      min_price: params.get('min_price') || undefined,
      max_price: params.get('max_price') || undefined,
      in_stock: params.get('in_stock') === 'true',
      set: params.get('set') || undefined,
      rarity: params.get('rarity') || undefined,
      conditions: params.get('conditions')?.split(',').filter(c => c) || [],
      sort: params.get('sort') || 'date_desc',
    });
  }, []);

  // Check if any filters are active
  const hasActiveFilters = useCallback(() => {
    return (
      query ||
      filters.min_price ||
      filters.max_price ||
      filters.in_stock ||
      filters.set ||
      filters.rarity ||
      (filters.conditions && filters.conditions.length > 0) ||
      filters.sort !== 'date_desc'
    );
  }, [query, filters]);

  return {
    // Current state
    query,
    debouncedQuery,
    filters,
    
    // Actions
    setSearchQuery,
    updateFilter,
    updateFilters,
    clearSearch,
    
    // Pagination
    goToPage,
    nextPage,
    prevPage,
    
    // Specific filters
    setPriceRange,
    toggleInStockFilter,
    setSortOption,
    
    // URL handling
    getSearchParams,
    loadFromSearchParams,
    
    // Helpers
    hasActiveFilters,
  };
}

// Hook for managing search suggestions state
export function useSearchSuggestions() {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const openSuggestions = useCallback(() => setIsOpen(true), []);
  const closeSuggestions = useCallback(() => {
    setIsOpen(false);
    setSelectedIndex(-1);
  }, []);

  const selectNext = useCallback((maxIndex) => {
    setSelectedIndex(prev => (prev < maxIndex - 1 ? prev + 1 : -1));
  }, []);

  const selectPrev = useCallback((maxIndex) => {
    setSelectedIndex(prev => (prev > -1 ? prev - 1 : maxIndex - 1));
  }, []);

  const selectSuggestion = useCallback((index) => {
    setSelectedIndex(index);
  }, []);

  return {
    isOpen,
    selectedIndex,
    openSuggestions,
    closeSuggestions,
    selectNext,
    selectPrev,
    selectSuggestion,
  };
} 