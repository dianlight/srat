import { Line } from 'react-chartjs-2';

export const renderConnectionsGraph = (connectionsData) => {
    const data = {
        labels: connectionsData.map((dataPoint) => dataPoint.label),
        datasets: [{
            label: 'Connections',
            data: connectionsData.map((dataPoint) => dataPoint.value),
            borderColor: 'rgba(75,192,192,1)',
            backgroundColor: 'rgba(75,192,192,0.2)',
            fill: true,
        }],
    };

    return <Line data={data} />;
};

// Add this function call in the appropriate place in your render method where you want to display the graph.
// Example: {renderConnectionsGraph(connectionsData)}
