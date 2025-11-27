export type Coord = number[];
export interface GridCell {
    coord: Coord;
    value: any;
}
export interface GridState {
    id: string;
    dimensions: number[];
    defaultValue?: any;
    cells: GridCell[];
}
export interface ClaimResult {
    success: boolean;
    error?: string;
}
export interface CellUpdateEvent {
    type: string;
    gridId: string;
    coord?: Coord;
    value?: any;
    [key: string]: any;
}
export interface GridClientOptions {
    baseUrl: string;
    gridId: string;
    userId?: string;
}
export interface GridClient {
    getInitialState(): Promise<GridState>;
    connect(): Promise<void>;
    disconnect(): void;
    isConnected(): boolean;
    claim(coord: Coord, value: any): Promise<ClaimResult>;
    release(coord: Coord): Promise<ClaimResult>;
    onCellUpdate(listener: (ev: CellUpdateEvent) => void): () => void;
}
export declare function createGridClient(options: GridClientOptions): GridClient;
