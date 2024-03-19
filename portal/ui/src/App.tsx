import { useEffect, useState } from 'react';

import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Paper from '@mui/material/Paper';
import Tooltip from '@mui/material/Tooltip';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Link from '@mui/material/Link';

import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/DeleteOutlined';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Close';
import FileCopyIcon from '@mui/icons-material/FileCopy';
import IconButton from '@mui/material/IconButton';
import { CopyToClipboard } from 'react-copy-to-clipboard';

import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import Avatar from '@mui/material/Avatar';
import ImageIcon from '@mui/icons-material/Image';
import WorkIcon from '@mui/icons-material/Work';
import BeachAccessIcon from '@mui/icons-material/BeachAccess';

import { Routes, Route } from './types';
import './App.css';
import { Typography } from '@mui/material';
import ImageMagic from './ImageMagic';

function App(): JSX.Element {
  const [copied, setCopied] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const [error, setError] = useState<string | null>(null);

  const [routes, setRoutes] = useState<Routes>([]);
  const [newRoute, setNewRoute] = useState<Route | null>(null);
  const [editableRoute, setEditableRoute] = useState<Route>({} as Route);

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
    return <div>Loading...</div>;
  }

  const handleAddNewRoute = () => {
    setNewRoute({
      key: '',
      url: ''
    });
  }

  const handleCancelAddNewRoute = () => {
    setError(null);
    setNewRoute(null);
  }

  const handleSaveAddNewRoute = () => {
    const err = validateRoute(newRoute);
    if (err) {
      setError(err);
      return;
    }


    setRoutes([...routes, newRoute as Route]);
    setNewRoute(null);
  }

  const handleEditRoute = (route: Route) => () => {
    setEditableRoute(route);
  }

  const handleDeleteRoute = () => { }

  const handleCopy = () => {
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const hostname = window.location.hostname;
  return (
    <>
      <Typography variant="h1" component="h1" gutterBottom>
        bridge
      </Typography>
      <List sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
        gap: '16px',
      }}>
        {routes.map((route: Route) => {
          const truncatedUrl = route.url.length > 23 ? route.url.substring(0, 23) + '...' : route.url;
          const clipboardText = hostname + route.key;
          return (
            <ListItem key={route.key}>
              <ListItemAvatar>
                <ImageMagic url={clipboardText} />
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

