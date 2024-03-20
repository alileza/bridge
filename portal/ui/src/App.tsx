import React, { useEffect, useState } from 'react';
import Tooltip from '@mui/material/Tooltip';
import Link from '@mui/material/Link';
import { CopyToClipboard } from 'react-copy-to-clipboard';
import CheckIcon from '@mui/icons-material/Check';
import Alert from '@mui/material/Alert';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import { Routes, Route } from './types';
import './App.css';
import { Typography } from '@mui/material';
import { TextField, Button } from '@mui/material';
import GitHubIcon from '@mui/icons-material/GitHub';

function App(): JSX.Element {
  const [copied, setCopied] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [routes, setRoutes] = useState<Routes>([]);
  const [newRoute, setNewRoute] = useState<Route>({ preview: '', key: '', url: '' });
  const [searchTerm, setSearchTerm] = useState<string>('');

  useEffect(() => {
    fetch('/api/routes')
      .then(res => res.json())
      .then((data: Routes) => {
        setRoutes(data);
      })
      .catch(console.error).finally(() => {
        setIsLoading(false);
      })
  }, []);

  if (isLoading) {
    return <Typography variant='h1' >Loading...</Typography>;
  }

  const handleKeyChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let newValue = e.target.value;

    if (newValue === '' || (newValue.charAt(0) !== '/' && newValue !== '0')) {
      newValue = '/';
    }
    setNewRoute({ ...newRoute, key: newValue });
  }


  const handleURLChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setNewRoute({ ...newRoute, url: e.target.value });
  }

  const handleKeyDownOnURLInput = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSaveNewRoute();
    }
  }

  const handleCopy = () => {
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const handleSaveNewRoute = () => {
    fetch('/api/routes', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(newRoute)
    })
      .then((response: Response) => {
        if (response.status !== 202) {
          if (response.status === 500) {
            setError('Failed to save new route: ' + response.statusText);
            return;
          }
          response.json().then(data => {
            setError('Failed to save new route: ' + response.statusText + ' => ' + data.error);
          });
          return;
        }
        
        newRoute.key = window.location.hostname + newRoute.key;
        if (routes && routes.length > 0) {
          setRoutes([newRoute, ...routes]);
        } else {
          setRoutes([newRoute]);
        }
        setNewRoute({ preview: '', key: '', url: '' });
        setMessage(`Route ${newRoute.key} is successfully added!`);
      })
      .catch(console.error);
  }

  const filteredRoutes = routes?.filter(route => route.key.includes(searchTerm));

  return (
    <>
      <Link href="https://github.com/alileza/bridge" sx={{ display: 'inline-block', color: 'black', textDecoration: 'none' }} target="_blank">
        <img src="/bridge.png" className="logo" width="80" style={{ marginRight: '10px', float: 'left' }} />
        <Typography variant="h2" style={{ float: 'left'}} component="h2">
          bridge
        </Typography>
        <GitHubIcon fontSize="large" style={{ marginTop: '20px', marginLeft: '20px' }} />
      </Link>
      
      <div style={{float: 'right'}}>

      <TextField
          id="search-bar"
          label="Search"
          variant="outlined"
          size="small"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          style={{  marginTop: '20px', marginRight: '20px' }}
          fullWidth
        />
        <br/>

        <TextField
          id="key"
          label="/<smthng-shrt>"
          variant="outlined"
          size="small"
          value={newRoute.key}
          onChange={handleKeyChange}
          style={{ marginTop: '20px', marginRight: '10px'}}
        />

        <TextField
          id="url"
          label="Destination URL (http://...)"
          variant="outlined"
          size="small"
          value={newRoute.url}
          onChange={handleURLChange}
          onKeyDown={handleKeyDownOnURLInput}
          style={{ marginTop: '20px', marginRight: '10px' }} />

        <Button
          variant="contained"
          onClick={handleSaveNewRoute}
          style={{ marginTop: '20px' }}
        >
          Save
        </Button>

      </div>


      <div style={{ clear: 'both' }}></div>
      <br/>
      {error &&
        <Alert onClick={() => setError(null)} icon={<ErrorOutlineIcon fontSize="inherit" />} severity="error">
          {error}
        </Alert>
      }
      {message &&
        <Alert onClick={() => setMessage(null)} icon={<CheckIcon fontSize="inherit" />} severity="success">
          {message}
        </Alert>
      }

      <List sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
        gap: '16px',
      }}>
        {filteredRoutes && filteredRoutes.map((route: Route) => {
          const truncatedUrl = route.url.length > 23 ? route.url.substring(0, 23) + '...' : route.url;
          const clipboardText = route.key;
          return (
            <ListItem key={route.key}>
              <CopyToClipboard text={clipboardText} onCopy={handleCopy}>
                <Tooltip
                  placement="top"
                  sx={{ cursor: 'pointer' }}
                  title={copied ? `${clipboardText} is copied` : "copy to clipboard"}
                  enterTouchDelay={0}
                >
                  <ListItemText primary={route.key} secondary={truncatedUrl} />
                </Tooltip>
              </CopyToClipboard>
            </ListItem>
          );
        }
        )}
      </List>
    </>
  )
}

export default App;
