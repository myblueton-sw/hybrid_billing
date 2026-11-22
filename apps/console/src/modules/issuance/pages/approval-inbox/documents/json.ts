import { strictJSON as parse, StrictJSONError } from "../../../../../service/strict-json";
import { DocumentError } from "./error";
export function strictJSON(text: string, maximum = 100000): unknown {
  try { return parse(text, maximum); }
  catch (error) {
    if (error instanceof StrictJSONError) throw new DocumentError("invalid_document_response");
    throw error;
  }
}
