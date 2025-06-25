import axios from 'axios';

// Create axios instance with base configuration
const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for logging
api.interceptors.request.use(
  (config) => {
    console.log(`API Request: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    console.error('API Request Error:', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    console.error('API Response Error:', error.response?.data || error.message);
    
    // Handle common error cases
    if (error.response?.status === 404) {
      throw new Error('Resource not found');
    } else if (error.response?.status === 500) {
      throw new Error('Server error occurred');
    } else if (error.code === 'ECONNABORTED') {
      throw new Error('Request timeout');
    } else if (!error.response) {
      throw new Error('Network error - please check if the backend server is running');
    }
    
    throw error;
  }
);

// Card API endpoints
export const cardAPI = {
  // Get all cards with optional filters
  getAllCards: (filters = {}) => {
    const params = new URLSearchParams();
    
    // Add filters to params
    if (filters.page) params.append('page', filters.page);
    if (filters.page_size) params.append('page_size', filters.page_size);
    if (filters.min_price) params.append('min_price', filters.min_price);
    if (filters.max_price) params.append('max_price', filters.max_price);
    if (filters.in_stock) params.append('in_stock', 'true');
    if (filters.sort) params.append('sort', filters.sort);
    
    return api.get(`/cards?${params.toString()}`);
  },

  // Search cards with query and filters
  searchCards: (query, filters = {}) => {
    const params = new URLSearchParams();
    
    // Add query
    if (query) params.append('q', query);
    
    // Add filters
    if (filters.page) params.append('page', filters.page);
    if (filters.page_size) params.append('page_size', filters.page_size);
    if (filters.min_price) params.append('min_price', filters.min_price);
    if (filters.max_price) params.append('max_price', filters.max_price);
    if (filters.in_stock) params.append('in_stock', 'true');
    if (filters.set) params.append('set', filters.set);
    if (filters.rarity) params.append('rarity', filters.rarity);
    if (filters.conditions && filters.conditions.length > 0) {
      params.append('conditions', filters.conditions.join(','));
    }
    if (filters.sort) params.append('sort', filters.sort);
    
    return api.get(`/cards/search?${params.toString()}`);
  },

  // Get single card by ID
  getCard: (id) => {
    return api.get(`/cards/${id}`);
  },

  // Get search suggestions
  getSuggestions: (query, limit = 10) => {
    const params = new URLSearchParams();
    params.append('q', query);
    params.append('limit', limit.toString());
    
    return api.get(`/cards/suggestions?${params.toString()}`);
  },
};

// Scraping API endpoints
export const scrapeAPI = {
  // Start scraping
  start: (options = {}) => {
    return api.post('/scrape/start', {
      in_stock_only: options.inStockOnly || true,
      max_pages: options.maxPages || 0, // 0 means all pages
    });
  },

  // Get scraping status
  getStatus: () => {
    return api.get('/scrape/status');
  },

  // Stop scraping
  stop: () => {
    return api.post('/scrape/stop');
  },

  // Pause scraping
  pause: () => {
    return api.post('/scrape/pause');
  },

  // Resume scraping
  resume: () => {
    return api.post('/scrape/resume');
  },

  // Restart scraping (stop current job if running, then start new one)
  restart: (options = {}) => {
    return api.post('/scrape/restart', {
      in_stock_only: options.inStockOnly || true,
      max_pages: options.maxPages || 0, // 0 means all pages
    });
  },
};

// Statistics API endpoints
export const statsAPI = {
  // Get database and scraper statistics
  getStats: () => {
    return api.get('/stats');
  },

  // Get available sort options
  getSortOptions: () => {
    return api.get('/sort-options');
  },
};

// Database management API endpoints
export const databaseAPI = {
  // Reset database (clear all data)
  reset: () => {
    return api.delete('/database/reset');
  },
};

// Tracker API endpoints
export const trackerAPI = {
  // Get all trackers
  getTrackers: (filters = {}) => {
    const params = new URLSearchParams();
    
    // Add filters
    if (filters.page) params.append('page', filters.page);
    if (filters.page_size) params.append('page_size', filters.page_size);
    if (filters.status) params.append('status', filters.status);
    if (filters.in_stock !== undefined) params.append('in_stock', filters.in_stock.toString());
    if (filters.user_id) params.append('user_id', filters.user_id);
    
    return api.get(`/tracker/list?${params.toString()}`);
  },

  // Add new tracker
  addTracker: (trackerData) => {
    return api.post('/tracker/add', {
      url: trackerData.url,
      name: trackerData.name || '',
    });
  },

  // Get specific tracker
  getTracker: (id) => {
    return api.get(`/tracker/${id}`);
  },

  // Update tracker
  updateTracker: (id, updates) => {
    return api.put(`/tracker/${id}`, updates);
  },

  // Delete tracker
  deleteTracker: (id) => {
    return api.delete(`/tracker/${id}`);
  },

  // Manual check all trackers
  checkNow: () => {
    return api.post('/tracker/check-now');
  },

  // Get tracker system status
  getStatus: () => {
    return api.get('/tracker/status');
  },

  // Get tracker options (available statuses, etc.)
  getOptions: () => {
    return api.get('/tracker/options');
  },
};

// Health check
export const healthAPI = {
  check: () => {
    return api.get('/health');
  },
};

// Helper functions for common operations
export const apiHelpers = {
  // Format price for display
  formatPrice: (price) => {
    return new Intl.NumberFormat('ja-JP', {
      style: 'currency',
      currency: 'JPY',
    }).format(price);
  },

  // Format date for display
  formatDate: (dateString) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  },

  // Build card image URL (fallback if needed)
  getCardImageUrl: (card) => {
    if (card.image_url) {
      let imageUrl = card.image_url;
      
      // Replace {width} placeholder with actual width (200px)
      imageUrl = imageUrl.replace(/\{width\}/gi, '200');
      
      // If it's a relative URL, make it absolute
      if (imageUrl.startsWith('/')) {
        // Use our API proxy for images to avoid CORS issues
        return `/api/proxy-image?url=${encodeURIComponent(`https://torecacamp-pokemon.com${imageUrl}`)}`;
      }
      // Use our API proxy for external images
      return `/api/proxy-image?url=${encodeURIComponent(imageUrl)}`;
    }
    
    // Fallback to a placeholder image
    return '/images/card-placeholder.svg';
  },

  // Get stock status text and color
  getStockStatus: (stock) => {
    if (stock > 10) {
      return { 
        text: 'In Stock', 
        color: 'text-white font-bold', 
        bgColor: 'bg-green-600 dark:bg-green-500' 
      };
    } else if (stock > 0) {
      return { 
        text: `${stock} left`, 
        color: 'text-white font-bold', 
        bgColor: 'bg-orange-500 dark:bg-orange-400' 
      };
    } else {
      return { 
        text: 'Out of Stock', 
        color: 'text-white font-bold', 
        bgColor: 'bg-red-600 dark:bg-red-500' 
      };
    }
  },

  // Build search URL for external link
  getExternalCardUrl: (card) => {
    if (card.url) {
      // If it's a relative URL, make it absolute
      if (card.url.startsWith('/')) {
        return `https://torecacamp-pokemon.com${card.url}`;
      }
      return card.url;
    }
    return null;
  },
};

// Simplified API interface for common tracker operations
const apiInterface = {
  // Get all trackers (simplified interface)
  getTrackers: async (filters = {}) => {
    const response = await trackerAPI.getTrackers(filters);
    return response.data;
  },

  // Add tracker (simplified interface)
  addTracker: async (trackerData) => {
    const response = await trackerAPI.addTracker(trackerData);
    return response.data;
  },

  // Delete tracker (simplified interface)
  deleteTracker: async (id) => {
    const response = await trackerAPI.deleteTracker(id);
    return response.data;
  },

  // Check trackers now (simplified interface)
  checkTrackersNow: async () => {
    const response = await trackerAPI.checkNow();
    return response.data;
  },

  // Get tracker status (simplified interface)
  getTrackerStatus: async () => {
    const response = await trackerAPI.getStatus();
    return response.data;
  },

  // Get tracker stats (simplified interface)  
  getTrackerStats: async () => {
    const response = await statsAPI.getStats();
    return response.data?.tracker_stats || {};
  },

  // Include all other APIs
  cards: cardAPI,
  scrape: scrapeAPI,
  stats: statsAPI,
  database: databaseAPI,
  tracker: trackerAPI,
  health: healthAPI,
  helpers: apiHelpers,
};

// Export the simplified API interface
export { api, apiInterface };

// Create a unified API object that includes both raw axios and simplified interfaces
export const unifiedAPI = {
  // Simplified tracker functions (what useTracker expects)
  getTrackers: async (filters = {}) => {
    const response = await trackerAPI.getTrackers(filters);
    return response.data;
  },

  addTracker: async (trackerData) => {
    const response = await trackerAPI.addTracker(trackerData);
    return response.data;
  },

  deleteTracker: async (id) => {
    const response = await trackerAPI.deleteTracker(id);
    return response.data;
  },

  checkTrackersNow: async () => {
    const response = await trackerAPI.checkNow();
    return response.data;
  },

  getTrackerStatus: async () => {
    const response = await trackerAPI.getStatus();
    return response.data;
  },

  getTrackerStats: async () => {
    const response = await statsAPI.getStats();
    return response.data?.tracker_stats || {};
  },

  // Include raw APIs for other functionality
  cards: cardAPI,
  scrape: scrapeAPI,
  stats: statsAPI,
  database: databaseAPI,
  tracker: trackerAPI,
  health: healthAPI,
  helpers: apiHelpers,
};

export default unifiedAPI; 