import { useContext, useEffect, useState } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { InView, useInView } from 'react-intersection-observer';
import { a11yDark, a11yLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { useColorScheme } from '@mui/material/styles';
import { useGetSambaConfigQuery } from '../store/sratApi';

export function SmbConf() {
    const { mode, setMode } = useColorScheme();
    const { ref, inView, entry } = useInView({
        /* Optional options */
        threshold: 0,
    });
    const smbconfig = useGetSambaConfigQuery();

    return (
        <InView as="div" onChange={(inView) => { inView && smbconfig.isSuccess }}>
            <SyntaxHighlighter customStyle={{ fontSize: '0.7rem' }} language="ini" style={mode === 'light' ? a11yLight : a11yDark} wrapLines wrapLongLines>
                {smbconfig.data?.data || ''}
            </SyntaxHighlighter>
        </InView>
    );
}