export interface Route {
  preview: string;
  key: string;
  url: string;
  updated_by?: string;
  updated_at?: string;
}

export type Routes = Route[];

export interface AuditEntry {
  time: string;
  actor: string;
  actor_emails?: string[];
  action: 'create' | 'update' | 'delete';
  key: string;
  url?: string;
  previous_url?: string;
}

export interface Me {
  auth_enabled: boolean;
  authenticated: boolean;
  login?: string;
  emails?: string[];
}
