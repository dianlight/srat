import { useContext, useEffect, useState } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { apiContext } from '../Contexts';
import { InView, useInView } from 'react-intersection-observer';

export function SmbConf() {
    const api = useContext(apiContext);
    const [smbConf, setSmbConf] = useState<string>('')
    const { ref, inView, entry } = useInView({
        /* Optional options */
        threshold: 0,
    });

    function updateSmbConf() {
        api.samba.sambaList().then((res) => {
            setSmbConf(res.data)
        }).catch(err => {
            console.error(err);
        })
    }

    useEffect(() => {
        updateSmbConf();
    }, []);

    return <InView as="div" onChange={(inView, entry) => { inView && updateSmbConf() }}>
        <SyntaxHighlighter language="ini">
            {smbConf}
        </SyntaxHighlighter>
    </InView>

}