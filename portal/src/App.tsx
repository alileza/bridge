import { useEffect, useState } from 'react';
import './App.css';

interface Config {
  [key: string]: string;
}

function App(): JSX.Element {
  const [config, setConfig] = useState<Config>({});
  const [editableKey, setEditableKey] = useState<string | null>(null);
  const [editableValue, setEditableValue] = useState<string | null>(null);
  const [newKey, setNewKey] = useState<string>('');
  const [newValue, setNewValue] = useState<string>('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/routes.json')
      .then((response) => response.json())
      .then((config) => {
        setConfig(config);
      });
  }, []);

  const handleEdit = (key: string): void => {
    setEditableKey(key);
    setEditableValue(config[key]);
  };

  const handleInputChange = (value: string): void => {
    setEditableValue(value);
  };

  const handleCancelEdit = (): void => {
    setEditableKey(null);
    setEditableValue(null);
  };

  const validateUrl = (url: string): boolean => {
    // Regular expression for URL validation
    const urlPattern = /^(ftp|http|https):\/\/[^ "]+$/;
    return urlPattern.test(url);
  };

  const handleSave = (key: string): void => {
    if (!validateUrl(editableValue || '')) {
      setError('Please enter a valid URL');
      return;
    }

    setConfig((prevConfig) => ({
      ...prevConfig,
      [key]: editableValue || '',
    }));

    fetch('/routes.json', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        path: key,
        target: editableValue,
      }),
    })
      .then(() => {
        setEditableKey(null);
        setEditableValue(null);
        setError(null);
      })
      .catch(() => {
        // Handle error if needed
      });
  };

  const handleAdd = (): void => {
    if (!validateUrl(newValue || '')) {
      setError('Please enter a valid URL');
      return;
    }

    setConfig((prevConfig) => ({
      ...prevConfig,
      [newKey]: newValue,
    }));

    setNewKey('');
    setNewValue('');

    fetch('/routes.json', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        path: newKey,
        target: newValue,
      }),
    })
      .then(() => {
        setError(null);
        // Handle success if needed
      })
      .catch(() => {
        // Handle error if needed
      });
  };

  return (
    <>
      <h1>trde.link</h1>
      <div className="error">{error}</div>
      <table>
        <thead>
          <tr>
            <th>Path</th>
            <th>Target URL</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {Object.keys(config).map((key) => (
            <tr key={key}>
              <td>{key}</td>
              <td>
                {editableKey === key ? (
                  <input
                    type="text"
                    value={editableValue || ''}
                    onChange={(e) => handleInputChange(e.target.value)}
                  />
                ) : (
                  config[key]
                )}
              </td>
              <td>
                {editableKey === key ? (
                  <>
                    <button onClick={() => handleSave(key)}>Save</button>
                    <button onClick={handleCancelEdit}>Cancel</button>
                  </>
                ) : (
                  <button onClick={() => handleEdit(key)}>Edit</button>
                )}
              </td>
            </tr>
          ))}
          <tr>
            <td>
              <input
                type="text"
                value={newKey}
                onChange={(e) => setNewKey(e.target.value)}
                placeholder="New Key"
              />
            </td>
            <td>
              <input
                type="text"
                value={newValue}
                onChange={(e) => setNewValue(e.target.value)}
                placeholder="New Value"
              />
            </td>
            <td>
              <button onClick={handleAdd}>Add</button>
            </td>
          </tr>
        </tbody>
      </table>
    </>
  );
}

export default App;
