import { useEffect, useState } from 'react';

import Tooltip from '@mui/material/Tooltip';
import Link from '@mui/material/Link';

import { CopyToClipboard } from 'react-copy-to-clipboard';
import CheckIcon from '@mui/icons-material/Check';
import Alert from '@mui/material/Alert';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import ListItemAvatar from '@mui/material/ListItemAvatar';

import { Routes, Route } from './types';
import './App.css';
import { Typography } from '@mui/material';
import ImageMagic from './ImageMagic';

function App(): JSX.Element {
  const [copied, setCopied] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [routes, setRoutes] = useState<Routes>([]);
  const [newRoute, setNewRoute] = useState<Route>({ preview: '', key: '', url: '' });

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
    
    // Check if the value is empty or if it's not / and not empty
    if (newValue === '' || (newValue.charAt(0) !== '/' && newValue !== '0')) {
        // Set the input value to empty string
        newValue = '';
    } 

    // Update the state with the modified value
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
        
        fetch('/api/routes/preview?url=https://' + window.location.hostname + newRoute.key, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json'
          },
        }).then(res => res.json()).then((data: { image: string }) => {
          newRoute.preview =  "data:image/png;base64," +data.image;
          setRoutes([newRoute, ...routes]);
          setNewRoute({ preview: '', key: '', url: '' });
        })

        setMessage(`Route ${newRoute.key} is successfully added!`);
      })
      .catch(console.error);
  }

  const hostname = window.location.hostname;
  return (
    <>
      <Link href="https://github.com/alileza/bridge" sx={{ color: 'black', textDecoration: 'none' }} target="_blank">
        <Typography variant="h2" component="h2" gutterBottom>
          bridge
        </Typography>
      </Link>
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
        <ListItem key='new-placeholder'>
          <ListItemAvatar>
            <ImageMagic
              imageContent={''}
              routeKeyURL={newRoute.key + newRoute.url}
              handleSave={handleSaveNewRoute}
            />
          </ListItemAvatar>
          <ListItemText
            primary={<input
              value={newRoute.key}
              onChange={handleKeyChange}
              placeholder="/<edit me>" />}
            secondary={<input
              value={newRoute.url}
              onChange={handleURLChange}
              onKeyDown={handleKeyDownOnURLInput}
              placeholder="https://<edit me>" />} />


        </ListItem>
        {routes && routes.map((route: Route) => {
          const truncatedUrl = route.url.length > 23 ? route.url.substring(0, 23) + '...' : route.url;
          const clipboardText = "https://" + hostname + route.key;
          return (
            <ListItem key={route.key}>
              <ListItemAvatar>
                <ImageMagic handleSave={() => {}} imageContent={route.preview} routeKeyURL={clipboardText} />
              </ListItemAvatar>
              <CopyToClipboard text={clipboardText} onCopy={handleCopy}>
                <Tooltip placement="top" sx={{ cursor: 'pointer' }} title={copied ? `${clipboardText} is copied` : "copy to clipboard"}>
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

