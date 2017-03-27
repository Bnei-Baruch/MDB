import React, { Component } from 'react';
import Spinner from './Spinner';
import './Files.css';

class Files extends Component {
    constructor(props) {
        super(props);

        this.state = {
            // Should be eventually props.
            files: [],
            matching: -1,
            total: -1,

            loadingFiles: false,
            error: '',

            // Should be eventually state.
            showRemoveIcon: false,
            searchText: '',
        }
    }

    componentDidMount = () => {
        this.getFiles('');
    }

    handleSearchChange = (e) => {
        this.getFiles(e.target.value);
    }

    handleSearchCancel = () => {
        this.getFiles('');
    }

    getFiles = (searchText) => {
        this.setState({
            loadingFiles: true,
            searchText,
            showRemoveIcon: searchText !== '',
        });
        return fetch('http://rt-dev.kbb1.com:8080/admin/rest/files?query=' + searchText)
        .then((response) => {
            if (!response.ok) {
                throw Error('Error loading files, response not ok.');
            }
            this.setState({loadingFiles: false});
            return response.json().then(json => {
                if (json.status && json.status === 'ok') {
                    this.setState({
                        files: json.files,
                        matching: json.matching,
                        total: json.total,
                        error: '',
                    });
                } else {
                    throw Error('Error loading files, got bad status.');
                }
            });
        }).catch((e) => {
            this.setState({
                loadingFiles: false,
                error: 'Error loading files: ' + e
            });
        })
    }

    render() {
        const { showRemoveIcon, files } = this.state;
        const removeIconStyle = showRemoveIcon ? {} : { visibility: 'hidden' };

        const fileRows = files.map((file, idx) => (
            <tr key={idx}>
                <td>{file.uid}</td>
                <td>{file.name}</td>
                <td className='right aligned'>{file.file_created_at}</td>
            </tr>
        ));

        return (
            <div>
                <div>
                    <table className='ui selectable structured large table'>
                        <thead>
                            <tr>
                                <th colSpan='3'>
                                    <div className='ui fluid search flex-space-between-center'>
                                        <div>
                                            <div className='ui icon input'>
                                                <input
                                                    className='prompt'
                                                    type='text'
                                                    placeholder='Search files...'
                                                    value={this.state.searchText}
                                                    onChange={this.handleSearchChange}
                                                />
                                                <i className='search icon' />
                                            </div>
                                            <i
                                                className='remove icon'
                                                onClick={this.handleSearchCancel}
                                                style={removeIconStyle}
                                            />
                                        </div>
                                        <div className='flex-space-between-center'>
                                            {this.state.loadingFiles ?
                                                <span className='flex-space-between-center'>
                                                    <Spinner/>
                                                    <span style={{marginLeft: '10px'}}>Searching...</span>
                                                </span> : null}
                                            {!!this.state.error ?
                                                <span style={{color: 'red', marginLeft: '10px'}}>{this.state.error}</span> : null}
                                        </div>
                                        <div>
                                            {this.state.matching >= 0 && this.state.total >= 0 ?
                                                <span>Matched {this.state.matching} of {this.state.total}</span> : null}
                                        </div>
                                    </div>
                                </th>
                            </tr>
                            <tr>
                                <th>UID</th>
                                <th className='eight wide'>Name</th>
                                <th>created at</th>
                            </tr>
                        </thead>
                        <tbody>
                            {fileRows}
                        </tbody>
                    </table>
                </div>
            </div>
        );
    }

}

export default Files;
