import path from 'path';

export function getDevtoolData() {
    const projectRoot = process.env.HOST_PROJECT_PATH || path.resolve();
    const jsonData = {
       workspace: {
           root: projectRoot,
           uuid: 'srat-uuid-bogus',
       }
    };
    return jsonData;
};