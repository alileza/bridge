import { useEffect, useState } from 'react';

import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Paper from '@mui/material/Paper';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';

import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/DeleteOutlined';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Close';

import './App.css';

interface Route {
  key: string;
  url: string;
}

type Routes = Route[];

function validateRoute(route: Route | null): string | null {
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

type EditableRowProps = {
  data: Route;

  handleSaveAddNewRoute: () => void;
  handleCancelAddNewRoute: () => void;
}

function EditableRow({ handleSaveAddNewRoute, handleCancelAddNewRoute, data }: EditableRowProps): JSX.Element {
  const [route, setRoute] = useState<Route>(data);

  const handlePathChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setRoute({ ...route, key: e.target.value });
  }

  const handleURLChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setRoute({ ...route, url: e.target.value });
  }

  return (
    <TableRow>
      <TableCell component="th" scope="row">
        <TextField fullWidth label="Key" onChange={handlePathChange} value={route.key} placeholder="Key path such as (/hello, or private.tr/hello (if you have custom domain))" variant="outlined" />
      </TableCell>
      <TableCell align="left">
        <TextField fullWidth label="Target URL" onChange={handleURLChange} value={route.url} placeholder="Target URL like (https://google.com)" variant="outlined" />
      </TableCell>
      <TableCell align="left">
        <Button color="success">
          <SaveIcon onClick={handleSaveAddNewRoute} />
        </Button>
        <Button color="error">
          <CancelIcon onClick={handleCancelAddNewRoute} />
        </Button>
      </TableCell>
    </TableRow>
  );
}

function App(): JSX.Element {
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

  return (
    <>
      <TableContainer component={Paper}>
        <Table sx={{ minWidth: 650 }} aria-label="simple table">
          <TableHead>
            <TableRow>
              <TableCell colSpan={4}>
                <Button color="primary" startIcon={<AddIcon />} onClick={handleAddNewRoute}>
                  Add a new route
                </Button>
                {error && <div>{error}</div>}
              </TableCell>
            </TableRow>
            <TableRow>
              <TableCell>Key</TableCell>
              <TableCell align="left">URL</TableCell>
              <TableCell align="left">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <EditableRow 
                  data={{} as Route }
                  handleSaveAddNewRoute={handleSaveAddNewRoute} 
                  handleCancelAddNewRoute={handleCancelAddNewRoute} />

            {routes.map((route: Route) => {
              if (route.key === editableRoute.key) {
                return <EditableRow 
                      key={route.key}
                      data={route}
                      handleSaveAddNewRoute={handleSaveAddNewRoute} 
                      handleCancelAddNewRoute={() => { setEditableRoute({} as Route) }} 
                    />;
              }

              return (
                <TableRow
                  key={route.key}
                >
                  <TableCell component="th" scope="row">
                    {route.key}
                  </TableCell>
                  <TableCell align="left">{route.url}</TableCell>
                  <TableCell align="left">
                    <Button color="info" onClick={handleEditRoute(route)}>
                      <EditIcon />
                    </Button>
                    {/* <Button color="error" onClick={handleDeleteRoute}>
                      <DeleteIcon />
                    </Button> */}
                  </TableCell>
                </TableRow>);
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
}


export default App;
