import AddIcon from '@mui/icons-material/Add';
import SaveIcon from '@mui/icons-material/Save';

function ImageMagic({ imageContent, routeKeyURL, handleSave }: { imageContent: string, routeKeyURL: string, handleSave: () => void }): JSX.Element {
    
    if (imageContent === '') {
        if (routeKeyURL !== '') {
            return (
                <div>
                    <div style={{ cursor: 'pointer', height: '35px', width: '35px'}} onClick={handleSave}>
                        <SaveIcon color='success' fontSize='large' />
                    </div>
                </div>
            );
        }

        return (
        <div>
            <div style={{border: 'solid 1px', height: '35px', width: '35px'}} onClick={() => handlePreviewClick(imageContent)}>
                <AddIcon fontSize='large' />
            </div>
        </div>
        );
    }

    const handlePreviewClick = (imageContent: string) => {
        const newWindow = window.open('', '_blank', 'width=420,height=445');
        
        if (newWindow) {
            newWindow.document.write(`
            <center>
                <img src="${imageContent}" alt="Preview" width="400" height="400"/>
                <p>${routeKeyURL}</p>
            </center>
            `);
        } else {
            console.error('Failed to open preview window. Please make sure pop-ups are allowed for this site.');
        }
    };
   

    return (
        <div>
            <div style={{cursor: 'pointer'}} onClick={() => handlePreviewClick(imageContent)}>
                <img src={imageContent} alt="Image" width="35" />
            </div>
        </div>
    );
}

export default ImageMagic;
