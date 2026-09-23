import React, { useCallback, useEffect, useState } from 'react';
import Alert from '@mui/material/Alert';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Link from '@mui/material/Link';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import {
  Check as CheckIcon,
  ContentCopy as ContentCopyIcon,
  DeleteOutline as DeleteOutlineIcon,
  ErrorOutline as ErrorOutlineIcon,
  GitHub as GitHubIcon,
  Logout as LogoutIcon,
  Search as SearchIcon,
  Star as StarIcon,
} from '@mui/icons-material';
import { AuditEntry, Me, Routes, Route } from './types';
import './App.css';

const emptyRoute: Route = { preview: '', key: '', url: '' };

// Route keys are stored as "<host>/<path>"; show only the path part.
const shortKey = (key: string) => key.slice(key.indexOf('/'));

const timeAgo = (iso: string) => {
  const seconds = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  const units: [number, string][] = [[31536000, 'y'], [2592000, 'mo'], [86400, 'd'], [3600, 'h'], [60, 'm']];
  for (const [size, unit] of units) {
    if (seconds >= size) return `${Math.floor(seconds / size)}${unit} ago`;
  }
  return 'just now';
};

// api wraps fetch and sends the user to GitHub login when the session is missing.
const api = async (input: string, init?: RequestInit) => {
  const response = await fetch(input, init);
  if (response.status === 401) {
    window.location.href = '/auth/login';
  }
  return response;
};

const actionLabel: Record<AuditEntry['action'], string> = {
  create: 'created',
  update: 'updated',
  delete: 'deleted',
};

function App(): JSX.Element {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [routes, setRoutes] = useState<Routes>([]);
  const [newRoute, setNewRoute] = useState<Route>(emptyRoute);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [me, setMe] = useState<Me | null>(null);
  const [view, setView] = useState<'routes' | 'activity'>('routes');
  const [activity, setActivity] = useState<AuditEntry[]>([]);

  useEffect(() => {
    api('/api/routes')
      .then(res => res.json())
      .then((data: Routes | null) => setRoutes(Array.isArray(data) ? data : []))
      .catch(console.error)
      .finally(() => setIsLoading(false));
    fetch('/api/me')
      .then(res => res.json())
      .then(setMe)
      .catch(console.error);
  }, []);

  const loadActivity = useCallback(() => {
    api('/api/audit?limit=200')
      .then(res => res.json())
      .then((data: AuditEntry[]) => setActivity(Array.isArray(data) ? data : []))
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (view === 'activity') loadActivity();
  }, [view, loadActivity]);

  const handleLogout = () => {
    fetch('/auth/logout', { method: 'POST' }).finally(() => window.location.reload());
  };

  const handleKeyChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let newValue = e.target.value;
    if (newValue === '' || newValue.charAt(0) !== '/') {
      newValue = '/' + newValue.replace(/^\/*/, '');
    }
    setNewRoute({ ...newRoute, key: newValue });
  };

  const handleCopy = (key: string) => {
    navigator.clipboard.writeText(key).catch(console.error);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 1500);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    api('/api/routes', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newRoute),
    })
      .then(async (response: Response) => {
        if (response.status !== 202) {
          const data = await response.json().catch(() => null);
          setError(`Failed to save route: ${data?.error ?? response.statusText}`);
          return;
        }
        const saved = {
          ...newRoute,
          key: window.location.host + newRoute.key,
          updated_by: me?.login ?? 'anonymous',
          updated_at: new Date().toISOString(),
        };
        setRoutes([saved, ...routes.filter(r => r.key !== saved.key)]);
        setNewRoute(emptyRoute);
        setMessage(`${saved.key} is ready to use`);
      })
      .catch(console.error);
  };

  const handleDelete = (key: string) => {
    if (!window.confirm(`Delete ${key}?`)) {
      return;
    }
    api('/api/routes', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key }),
    })
      .then((response: Response) => {
        if (!response.ok) {
          setError(`Failed to delete ${key}: ${response.statusText}`);
          return;
        }
        setRoutes(routes.filter(r => r.key !== key));
      })
      .catch(console.error);
  };

  const term = searchTerm.toLowerCase();
  const filteredRoutes = routes
    .filter(route => route.key.toLowerCase().includes(term) || route.url.toLowerCase().includes(term))
    .sort((a, b) => a.key.localeCompare(b.key));

  return (
    <Box className="container">
      <header className="header">
        <Link href="https://github.com/alileza/bridge" target="_blank" rel="noreferrer" className="brand" underline="none">
          <img src="/bridge.png" alt="" width="56" height="56" />
          <Typography variant="h2" component="h1" className="brand-name" sx={{ fontFamily: "Baltore, sans-serif" }}>bridge</Typography>
        </Link>
        <Link href="https://github.com/alileza/bridge" target="_blank" rel="noreferrer" className="github" underline="none" aria-label="Star bridge on GitHub">
          <GitHubIcon />
          <StarIcon className="star" />
        </Link>
        <Box component="form" className="new-route" onSubmit={handleSave}>
          <TextField
            id="key"
            label="Short path"
            placeholder="/smthng-shrt"
            variant="standard"
            value={newRoute.key}
            onChange={handleKeyChange}
            InputProps={{
              startAdornment: <InputAdornment position="start">{window.location.host}</InputAdornment>,
            }}
            className="key-input"
            required
          />
          <TextField
            id="url"
            label="Destination URL"
            placeholder="https://..."
            variant="standard"
            type="url"
            value={newRoute.url}
            onChange={e => setNewRoute({ ...newRoute, url: e.target.value })}
            className="url-input"
            required
          />
          <Button type="submit" variant="contained" disableElevation className="save">
            Save
          </Button>
        </Box>
        {me?.authenticated && me.login &&
          <Box className="user">
            <Avatar src={`https://github.com/${me.login}.png?size=64`} alt="" sx={{ width: 28, height: 28 }} />
            <Typography variant="body2">@{me.login}</Typography>
            <Tooltip title="Log out">
              <IconButton size="small" onClick={handleLogout} aria-label="Log out">
                <LogoutIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        }
      </header>


      {error &&
        <Alert onClose={() => setError(null)} icon={<ErrorOutlineIcon fontSize="inherit" />} severity="error" square>
          {error}
        </Alert>
      }
      {message &&
        <Alert onClose={() => setMessage(null)} icon={<CheckIcon fontSize="inherit" />} severity="success" square>
          {message}
        </Alert>
      }

      <Box className="list-header">
        <Box className="tabs" role="tablist">
          <button type="button" role="tab" aria-selected={view === 'routes'} className={view === 'routes' ? 'tab active' : 'tab'} onClick={() => setView('routes')}>
            {routes.length} {routes.length === 1 ? 'route' : 'routes'}
          </button>
          <button type="button" role="tab" aria-selected={view === 'activity'} className={view === 'activity' ? 'tab active' : 'tab'} onClick={() => setView('activity')}>
            Activity
          </button>
        </Box>
        {view === 'routes' &&
          <TextField
            id="search"
            placeholder="Search routes"
            variant="standard"
            size="small"
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment>,
            }}
          />
        }
      </Box>

      {view === 'activity' ? (
        activity.length === 0 ? (
          <Box className="empty"><Typography color="text.secondary">No activity recorded yet.</Typography></Box>
        ) : (
          <Box component="ol" className="activity">
            {activity.map((e, i) => (
              <Box component="li" key={`${e.time}-${i}`} className="activity-row">
                <Typography variant="body2" color="text.secondary" className="activity-time" title={new Date(e.time).toLocaleString()}>
                  {timeAgo(e.time)}
                </Typography>
                <Typography variant="body2" className="activity-text">
                  <strong>@{e.actor}</strong> {actionLabel[e.action]} <strong>{shortKey(e.key)}</strong>
                  {e.action === 'update' && <> <span className="muted">{e.previous_url}</span> → {e.url}</>}
                  {e.action === 'create' && <> → {e.url}</>}
                  {e.action === 'delete' && e.previous_url && <> <span className="muted">(was {e.previous_url})</span></>}
                </Typography>
              </Box>
            ))}
          </Box>
        )
      ) : isLoading ? (
        <Box className="empty"><CircularProgress color="inherit" size={28} /></Box>
      ) : filteredRoutes.length === 0 ? (
        <Box className="empty">
          <Typography color="text.secondary">
            {routes.length === 0 ? 'No routes yet. Create your first one above.' : 'No routes match your search.'}
          </Typography>
        </Box>
      ) : (
        <Box component="ul" className="routes">
          {filteredRoutes.map((route: Route) => (
            <Box component="li" key={route.key} className="route">
              <Box className="route-text">
                <Typography className="route-key" noWrap title={route.key}>{shortKey(route.key)}</Typography>
                <Link href={route.url} target="_blank" rel="noreferrer" className="route-url" noWrap title={route.url} color="text.secondary">
                  {route.url}
                </Link>
                {route.updated_by && route.updated_at &&
                  <Typography variant="caption" color="text.secondary" noWrap title={new Date(route.updated_at).toLocaleString()}>
                    @{route.updated_by} · {timeAgo(route.updated_at)}
                  </Typography>
                }
              </Box>
              <Tooltip title={copiedKey === route.key ? 'Copied!' : 'Copy short link'} placement="top">
                <IconButton size="small" onClick={() => handleCopy(route.key)} aria-label={`Copy ${route.key}`}>
                  {copiedKey === route.key ? <CheckIcon fontSize="small" /> : <ContentCopyIcon fontSize="small" />}
                </IconButton>
              </Tooltip>
              <Tooltip title="Delete" placement="top">
                <IconButton size="small" onClick={() => handleDelete(route.key)} aria-label={`Delete ${route.key}`}>
                  <DeleteOutlineIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
}

export default App;
