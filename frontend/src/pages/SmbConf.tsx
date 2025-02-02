import { useContext, useEffect, useState } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { apiContext as api } from '../Contexts';
import { InView, useInView } from 'react-intersection-observer';
import { a11yDark, a11yLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { useColorScheme } from '@mui/material/styles';

export function SmbConf() {
    const [smbConf, setSmbConf] = useState<string>('')
    const { mode, setMode } = useColorScheme();
    const { ref, inView, entry } = useInView({
        /* Optional options */
        threshold: 0,
    });

    function updateSmbConf() {
        api.samba.configList().then((res) => {
            setSmbConf(res.data.data || "No data available")
        }).catch(err => {
            console.error(err);
        })
    }

    useEffect(() => {
        updateSmbConf();
    }, []);

    return <InView as="div" onChange={(inView, entry) => { inView && updateSmbConf() }}>
        <SyntaxHighlighter customStyle={{ fontSize: '0.7rem' }} language="ini" style={mode === 'light' ? a11yLight : a11yDark} wrapLines wrapLongLines>
            {smbConf}
        </SyntaxHighlighter>
    </InView>

}