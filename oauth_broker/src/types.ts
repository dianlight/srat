export type ProviderConfig = {
  client_id: string;
  client_secret: string;
  authorize_url?: string;
  token_url?: string;
  scopes?: string[];
  auth_params?: Record<string, string>;
};

export type ProvidersConfig = Record<string, ProviderConfig>;

export type BrokerSession = {
  auth_url: string;
  session_id: string;
};

export type BrokerToken = {
  token_json: string;
  account_label?: string;
  client_id?: string;
  client_secret?: string;
};

export type SessionRecord = {
  provider: string;
  sratCallbackUrl: string;
  createdAt: number;
  // Instance binding (hardened flow): every session is tied to a registered instance
  instanceId?: string;
  // Set after provider callback completes token exchange
  tokenJson?: string;
  accountLabel?: string;
  clientId?: string;
  clientSecret?: string;
  completedAt?: number;
  // PKCE S256: verifier stored at /v1/start, consumed at /v1/callback token exchange
  codeVerifier?: string;
};

export type InstanceRecord = {
  instanceId: string;
  redirectUrl: string;
  createdAt: number;
};

export type HealthResponse = {
  status: "ok";
  providers: string[];
};
