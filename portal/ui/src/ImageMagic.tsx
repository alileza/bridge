import { useState, useEffect } from 'react';
import Paper from '@mui/material/Paper';


function ImageMagic({ url }: { url: string }): JSX.Element {
    const [imageContent, setImageContent] = useState<string>('');
    const [showPreview, setShowPreview] = useState<boolean>(false);

    useEffect(() => {
        fetch('/api/routes/preview?url=' + url)
            .then(res => res.json())
            .then((data: { image: string }) => { // Update the type of 'data' parameter
                setImageContent(data.image);
            })
            .catch(console.error);
    }, [url]);

    const handlePreviewClick = (imageContent: string) => {
        // Open a new window with the image and specified dimensions
        const newWindow = window.open('', '_blank', 'width=420,height=445');
        
        // Write the image content to the new window
        if (newWindow) {
            newWindow.document.write(`
            <center>
                <img src="data:image/png;base64,${imageContent}" alt="Preview" width="400" height="400"/>
                <p>${url}</p>
            </center>
            `);
        } else {
            console.error('Failed to open preview window. Please make sure pop-ups are allowed for this site.');
        }
    };


    return (
        <div>
            <div style={{cursor: 'pointer'}} onClick={() => handlePreviewClick(imageContent)}>
                <img src={"data:image/png;base64," + imageContent} alt="Image" width="35" />
            </div>
        </div>
    );
}

export default ImageMagic;
