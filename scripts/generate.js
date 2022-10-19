// This script generates Go code from the jsonschema
// bundle using handlebars templates. The templates are
// located in the ./templates subdirectory.

// This code is a work-in-progress.
// Please excuse the mess.

import fs from "fs";
import {exec} from "child_process";
import Case from "case";
import Handlebars from "handlebars";
import getType from "./helpers/types.js";

// parse the schema
let schema = JSON.parse(fs.readFileSync("dist/schema.json"));

// for each definition
Object.entries(schema.definitions).forEach(([k, v]) => {
    // name of the go file.
    const filename = v["x-file"].replace(".yaml", ".go");

    // store the go struct details.
    let struct = {
        name: k,
        desc: v.description,
        path: filename,
        props: [],
        types: [],
    }

    // for each property
    Object.entries(v.properties).forEach(([propkey, propval]) => {
        const json = propkey;
        const name = Case.pascal(propkey);

        // append the field to the go struct
        struct.props.push({
            name: name,
            json: json,
            type: getType(propval),
        });
    });

    // this block of code detects if we are using the type /
    // spec pattern. If yes, we store the type enum values
    // and their associated struct types.
    if (v.properties.type && v.properties.type.enum && v.oneOf) {
        v.oneOf.forEach(({allOf}) => {
            const name = allOf[0].properties.type.const;
            const type = allOf[1].properties.spec.$ref.slice(14);
            struct.types.push({
                name: name,
                type: type,
            })
        });
    }

    // parse the handlebars templates
    const text = fs.readFileSync("scripts/templates/struct.handlebars");
    const tmpl = Handlebars.compile(text.toString());
    
    // execute the template and write the contents
    // to the struct filepath.
    fs.writeFileSync(`dist/go/${struct.path}`, tmpl(struct)); 

    console.log(`dist/go/${struct.path}`)
});

// format generated files and ensure imports
exec("gofmt -s -w dist/go/*.go")
exec("goimports -w dist/go/*.go")
