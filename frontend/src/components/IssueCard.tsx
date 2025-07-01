import React from 'react';
import { Card, CardContent, Typography, Button, Box, Link } from '@mui/material';
import type { Issue } from '../store/sratApi';

interface IssueCardProps {
  issue: Issue;
  onResolve: (id: number) => void;
}

const IssueCard: React.FC<IssueCardProps> = ({ issue, onResolve }) => {
  return (
    <Card sx={{ mb: 2, backgroundColor: '#f8d7da', border: '1px solid #f5c6cb' }}>
      <CardContent>
        <Typography variant="h6" component="div">
          {issue.title}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {issue.description}
        </Typography>
        {issue.detailLink && (
          <Link href={issue.detailLink} target="_blank" rel="noopener" variant="body2">
            Details
          </Link>
        )}
        {issue.resolutionLink && (
          <Link href={issue.resolutionLink} target="_blank" rel="noopener" variant="body2" sx={{ ml: 1 }}>
            Resolution
          </Link>
        )}
        <Box sx={{ mt: 2 }}>
          <Button variant="contained" color="primary" onClick={() => onResolve(issue.id)}>
            Resolve
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
};

export default IssueCard;
