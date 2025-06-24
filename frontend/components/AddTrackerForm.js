import React, { useState } from 'react';

const AddTrackerForm = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    url: '',
    name: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value,
    }));
    
    // Clear error when user starts typing
    if (error) {
      setError('');
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!formData.url.trim()) {
      setError('URL is required');
      return;
    }

    // Basic URL validation
    try {
      new URL(formData.url);
    } catch {
      setError('Please enter a valid URL');
      return;
    }

    try {
      setIsSubmitting(true);
      setError('');
      
      await onSubmit(formData.url.trim(), formData.name.trim() || null);
      
      // Reset form on success
      setFormData({ url: '', name: '' });
    } catch (err) {
      console.error('Failed to add tracker:', err);
      setError(err.response?.data?.error || err.message || 'Failed to add tracker');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Escape') {
      onCancel();
    }
  };

  const handlePaste = (e) => {
    // If pasting into URL field and name is empty, try to extract name from URL
    if (e.target.name === 'url' && !formData.name) {
      setTimeout(() => {
        const url = e.target.value;
        try {
          const urlObj = new URL(url);
          const pathname = urlObj.pathname;
          
          // Try to extract a meaningful name from the URL path
          const segments = pathname.split('/').filter(Boolean);
          if (segments.length > 0) {
            const lastSegment = segments[segments.length - 1];
            // Remove file extensions and decode URI
            const name = decodeURIComponent(lastSegment.replace(/\.[^/.]+$/, ''));
            // Clean up the name (replace hyphens/underscores with spaces, capitalize)
            const cleanName = name
              .replace(/[-_]/g, ' ')
              .replace(/\b\w/g, l => l.toUpperCase());
            
            if (cleanName && cleanName.length > 0) {
              setFormData(prev => ({
                ...prev,
                name: cleanName,
              }));
            }
          }
        } catch {
          // Invalid URL, ignore
        }
      }, 100);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="p-3 bg-red-50 dark:bg-red-900 border border-red-200 dark:border-red-700 rounded-lg">
          <p className="text-red-800 dark:text-red-200 text-sm">{error}</p>
        </div>
      )}

      <div>
        <label 
          htmlFor="url" 
          className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
        >
          Product URL *
        </label>
        <input
          type="url"
          id="url"
          name="url"
          value={formData.url}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          placeholder="https://example.com/product-page"
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
          required
          disabled={isSubmitting}
        />
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Enter the product page URL you want to monitor for stock changes
        </p>
      </div>

      <div>
        <label 
          htmlFor="name" 
          className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
        >
          Display Name (Optional)
        </label>
        <input
          type="text"
          id="name"
          name="name"
          value={formData.name}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder="e.g., Charizard VMAX - Brilliant Stars"
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
          disabled={isSubmitting}
        />
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Give this tracker a memorable name (will auto-generate if left empty)
        </p>
      </div>

      <div className="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={isSubmitting || !formData.url.trim()}
          className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isSubmitting ? (
            <span className="flex items-center justify-center gap-2">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              Adding...
            </span>
          ) : (
            'Add Tracker'
          )}
        </button>
        
        <button
          type="button"
          onClick={onCancel}
          disabled={isSubmitting}
          className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 disabled:opacity-50 transition-colors"
        >
          Cancel
        </button>
      </div>

      <div className="bg-blue-50 dark:bg-blue-900 border border-blue-200 dark:border-blue-700 rounded-lg p-3">
        <h4 className="text-sm font-medium text-blue-800 dark:text-blue-200 mb-1">
          💡 Tips:
        </h4>
        <ul className="text-xs text-blue-700 dark:text-blue-300 space-y-1">
          <li>• Make sure the URL is a direct link to the product page</li>
          <li>• The system will automatically detect stock status changes</li>
          <li>• You can edit the display name after adding the tracker</li>
          <li>• Monitoring will start automatically once you add trackers</li>
        </ul>
      </div>
    </form>
  );
};

export default AddTrackerForm; 