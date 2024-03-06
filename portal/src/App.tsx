  import { useEffect, useState } from 'react';
  import { Table, Input, Button, Space, ConfigProvider } from 'antd';
  // import 'antd/dist/antd.dark.css';
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

    const columns = [
      {
        title: 'Path',
        dataIndex: 'path',
        key: 'path',
      },
      {
        title: 'Target URL',
        dataIndex: 'target',
        key: 'target',
        render: (text: string, record: any) => (
          editableKey === record.key ? (
            <Input
              value={editableValue || ''}
              onChange={(e) => handleInputChange(e.target.value)}
            />
          ) : (
            text
          )
        ),
      },
      {
        title: 'Action',
        key: 'action',
        render: (_text: string, record: any) => (
          editableKey === record.key ? (
            <Space>
              <Button type="primary" onClick={() => handleSave(record.key)}>Save</Button>
              <Button onClick={handleCancelEdit}>Cancel</Button>
            </Space>
          ) : (
            <Button onClick={() => handleEdit(record.key)}>Edit</Button>
          )
        ),
      },
    ];

    const dataSource = Object.keys(config).map((key) => ({
      key,
      path: key,
      target: config[key],
    }));

    return (
      <ConfigProvider>
        <div style={{ padding: "0px 30px" }}>
          <h1 className="title is-1">trde.link</h1>
          <h5 className="subtitle is-5">powered by <a href="https://github.com/alileza/bridge">bridge</a></h5>
          <div className="error">{error}</div>
          <Table
            columns={columns}
            dataSource={dataSource}
            pagination={false}
            bordered
            style={{ marginTop: '20px' }}
          />
          <div style={{ marginTop: '20px' }}>
            <Input
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              placeholder="Path Name (e.g. /example)"
              style={{ width: '200px', marginRight: '10px' }}
            />
            <Input
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              placeholder="Target URL (e.g. https://example.com)"
              style={{ width: '300px', marginRight: '10px' }}
            />
            <Button type="primary" onClick={handleAdd}>Add</Button>
          </div>
        </div>
      </ConfigProvider>
    );
  }

  export default App;
