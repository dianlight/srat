import React from 'react';
import { Card, CardContent, Typography, Button, Box, Link } from '@mui/material';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import VisibilityIcon from '@mui/icons-material/Visibility';
import type { Issue } from '../store/sratApi';
import { useIgnoredIssues } from '../hooks/issueHooks';

interface IssueCardProps {
  issue: Issue;
  onResolve: (id: number) => void;
  showIgnored?: boolean;
}

const IssueCard: React.FC<IssueCardProps> = ({ issue, onResolve, showIgnored }) => {
  const { isIssueIgnored, ignoreIssue, unignoreIssue } = useIgnoredIssues();
  const isIgnored = isIssueIgnored(issue.id);

  // When showIgnored is false, show only non-ignored items
  // When showIgnored is true, show all items
  if (!showIgnored && isIgnored) {
    return null;
  }

  return (
    <Card sx={{ mb: 2 }}>
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
        <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
          <Button variant="contained" color="primary" onClick={() => onResolve(issue.id)}>
            Resolve
          </Button>
          <Button
            variant="outlined"
            startIcon={isIgnored ? <VisibilityIcon /> : <VisibilityOffIcon />}
            onClick={() => isIgnored ? unignoreIssue(issue.id) : ignoreIssue(issue.id)}
          >
            {isIgnored ? 'Unignore' : 'Ignore'}
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
};

export default IssueCard;
