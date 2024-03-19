export interface Route {
    key: string;
    url: string;
  }
  
  export type Routes = Route[];


export function validateRoute(route: Route | null): string | null {
    if (!route) {
      return 'Route is required';
    }
  
    if (!route.key || !route.key.includes('/')) {
      return 'Key is required and must include a "/"';
    }
    if (!route.url) {
      return 'URL is required';
    }
    // Validate URL pattern here
    const urlPatternRegex = /^(https?|ftp):\/\/[^\s/$.?#].[^\s]*$/i;
    if (!urlPatternRegex.test(route.url)) {
      return 'Invalid URL format';
    }
  
    return null;
  }
  