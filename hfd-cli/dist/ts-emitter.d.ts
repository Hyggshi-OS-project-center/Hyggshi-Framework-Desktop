export interface SchemaField {
    name: string;
    type: SchemaNode;
}
export type SchemaNode = {
    /** named type / struct name (absent on composites) */
    name?: string;
    pkg?: string;
    /** struct | array | ptr | map | string | number | bool | … */
    kind: string;
    /** struct only */
    fields?: SchemaField[];
    /** array / ptr only */
    elem?: SchemaNode;
    /** map only */
    key?: SchemaNode;
    /** map only */
    val?: SchemaNode;
};
export interface SchemaBinding {
    name: string;
    in: SchemaNode;
    out: SchemaNode;
}
export interface Schema {
    bindings?: SchemaBinding[];
    error?: string;
}
export interface EmittedBindings {
    dts: string;
    js: string;
}
export declare function emitBindings(schema: Schema): EmittedBindings;
