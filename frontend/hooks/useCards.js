import useSWR from 'swr';
import { cardAPI } from '../lib/api';

// Fetcher function for SWR
const fetcher = async ([endpoint, query, filters]) => {
  // Check if we have any active filters
  const hasActiveFilters = query || 
    filters.min_price || 
    filters.max_price || 
    filters.in_stock || 
    filters.set || 
    filters.rarity || 
    (filters.conditions && filters.conditions.length > 0) ||
    (filters.page && filters.page > 1) ||
    (filters.sort && filters.sort !== 'date_desc');

  if (hasActiveFilters) {
    // Use search endpoint when any filters or query are active
    const response = await cardAPI.searchCards(query || '', filters);
    return response.data;
  } else {
    // Use getAllCards only when no filters are active
    const response = await cardAPI.getAllCards(filters);
    return response.data;
  }
};

// Custom hook for fetching and managing cards
export function useCards(searchQuery = '', filters = {}) {
  const {
    data,
    error,
    isLoading,
    mutate,
    isValidating
  } = useSWR(
    ['cards', searchQuery, filters],
    fetcher,
    {
      // Revalidate on focus
      revalidateOnFocus: false,
      // Revalidate on reconnect
      revalidateOnReconnect: true,
      // Dedupe requests within 5 seconds
      dedupingInterval: 5000,
      // Keep data for 5 minutes
      focusThrottleInterval: 300000,
      // Error retry
      errorRetryCount: 3,
      errorRetryInterval: 1000,
    }
  );

  return {
    // Data
    cards: data?.cards || [],
    totalPages: data?.total_pages || 0,
    totalItems: data?.total || 0,
    currentPage: data?.page || 1,
    pageSize: data?.page_size || 20,
    
    // State
    isLoading,
    isValidating,
    error,
    isEmpty: !isLoading && (!data?.cards || data.cards.length === 0),
    
    // Actions
    refresh: mutate,
    
    // Helper methods
    hasNextPage: () => {
      const totalPages = data?.total_pages || 0;
      const currentPage = data?.page || 1;
      return currentPage < totalPages;
    },
    
    hasPrevPage: () => {
      const currentPage = data?.page || 1;
      return currentPage > 1;
    },
  };
}

// Hook for getting a single card
export function useCard(cardId) {
  const {
    data,
    error,
    isLoading,
    mutate
  } = useSWR(
    cardId ? `card-${cardId}` : null,
    () => cardAPI.getCard(cardId).then(res => res.data),
    {
      revalidateOnFocus: false,
      errorRetryCount: 3,
    }
  );

  return {
    card: data,
    isLoading,
    error,
    refresh: mutate,
  };
}

// Hook for search suggestions
export function useSearchSuggestions(query, limit = 10) {
  const {
    data,
    error,
    isLoading
  } = useSWR(
    query && query.length > 2 ? ['suggestions', query, limit] : null,
    () => cardAPI.getSuggestions(query, limit).then(res => res.data),
    {
      revalidateOnFocus: false,
      dedupingInterval: 2000,
      errorRetryCount: 1,
    }
  );

  return {
    suggestions: data?.suggestions || [],
    isLoading,
    error,
  };
} 