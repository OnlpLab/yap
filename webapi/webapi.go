package webapi

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/gorilla/mux"
	"github.com/OnlpLab/yap/nlp/format/lex"
	"github.com/OnlpLab/yap/nlp/types"
	"strings"
	"github.com/OnlpLab/yap/app"
	"github.com/OnlpLab/yap/nlp/parser/joint"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/conll"
)


var (
	router *mux.Router
)

type Request struct {
	Text string `json:text`
	AmbLattice string `json:amb_lattice`
	DisambLattice string `json:disamb_lattice`
}

type Data struct {
	MALattice string `json:"ma_lattice,omitempty"`
	MDLattice string `json:"md_lattice,omitempty"`
	DepTree string `json:"dep_tree,omitempty"`
	Error error `json:"error,omitempty"`
}

func HebrewMorphAnalyzerHandler(resp http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		data := Data { Error: err }
		respondWithJSON(resp, http.StatusBadRequest, data)
		return
	}
	rawText := strings.Replace(request.Text, " ", "\n", -1)
	maLattice := HebrewMorphAnalyzeRawSentences(rawText)
	data := Data{ MALattice: maLattice }
	respondWithJSON(resp, http.StatusOK, data)
}

func MorphDisambiguatorHandler(resp http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		data := Data { Error: err }
		respondWithJSON(resp, http.StatusBadRequest, data)
		return
	}
	ambLattice := strings.Replace(request.AmbLattice, "\\t", "\t", -1)
	ambLattice = strings.Replace(ambLattice, "\\n", "\n", -1)
	mdLattice := MorphDisambiguateLattices(ambLattice)
	data := Data { MDLattice: mdLattice }
	respondWithJSON(resp, http.StatusOK, data)
}

func DepParserHandler(resp http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		result := Data { Error: err }
		respondWithJSON(resp, http.StatusBadRequest, result)
		return
	}
	disambLattice := strings.Replace(request.DisambLattice, "\\t", "\t", -1)
	disambLattice = strings.Replace(disambLattice, "\\n", "\n", -1)
	depTree := DepParseDisambiguatedLattice(disambLattice)
	data := Data { DepTree: depTree }
	respondWithJSON(resp, http.StatusOK, data)
}

func HebrewPipelineHandler(resp http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		data := Data { Error: err }
		respondWithJSON(resp, http.StatusBadRequest, data)
		return
	}
	rawText := strings.Replace(request.Text, " ", "\n", -1)
	maLattice := HebrewMorphAnalyzeRawSentences(rawText)
	mdLattice := MorphDisambiguateLattices(maLattice)
	depTree := DepParseDisambiguatedLattice(mdLattice)
	data := Data { MALattice: maLattice, MDLattice: mdLattice, DepTree: depTree }
	respondWithJSON(resp, http.StatusOK, data)
}

func HebrewJointHandler(resp http.ResponseWriter, req *http.Request) {
	request := Request{}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		data := Data { Error: err }
		respondWithJSON(resp, http.StatusBadRequest, data)
		return
	}
	rawText := strings.Replace(request.Text, " ", "\n", -1)
	maLattice := HebrewMorphAnalyzeRawSentences(rawText)
	depTree, mdLattice, _ := JointParseAmbiguousLattices(maLattice)
	data := Data { MALattice: maLattice, MDLattice: mdLattice, DepTree: depTree }
	respondWithJSON(resp, http.StatusOK, data)
}

func respondWithJSON(resp http.ResponseWriter, code int, payload Data) {
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(code)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		errJson, _  := json.Marshal(err)
		resp.Write(errJson)
	} else {
		resp.Write(jsonPayload)
	}
}


func APIServerStartCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       StartAPIServer,
		UsageLine: "api",
		Short:     "start api server",
		Long: `
listen to api requests

	$ ./yap api [options]

`,
		Flag: *flag.NewFlagSet("api", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&app.HebMaPrefixFile, "ma_prefix", "bgupreflex_withdef.utf8.hr", "Prefix file for morphological analyzer")
	cmd.Flag.StringVar(&app.HebMaLexiconFile, "ma_lexicon", "bgulex.utf8.hr", "Lexicon file for morphological analyzer")
	cmd.Flag.BoolVar(&app.HebMaAlwaysnnp, "ma_always_nnp", false, "Always add NNP to tokens and prefixed subtokens")
	cmd.Flag.BoolVar(&app.HebMaNnpnofeats, "ma_add_nnp_no_feats", false, "Add NNP in lex but without features")
	cmd.Flag.BoolVar(&app.HebMaShowoov, "ma_show_oov", false, "Output OOV tokens")
	cmd.Flag.BoolVar(&lex.LOG_FAILURES, "ma_show_lex_error", false, "Log errors encountered when loading the lexicon")
	cmd.Flag.IntVar(&app.BeamSize, "beam", 64, "Beam size")
	cmd.Flag.BoolVar(&app.UsePOP, "use_end_token", true, "Use end token (pop)")
	cmd.Flag.BoolVar(&lattice.IGNORE_LEMMA, "nolemma", true, "Ignore lemmas")
	//cmd.Flag.BoolVar(&conll.IGNORE_LEMMA, "conll_nolemma", true, "Ignore lemmas")
	cmd.Flag.StringVar(&conll.WORD_TYPE, "conll_wordtype", "form", "Word type [form, lemma, lemma+f (=lemma if present else form)]")
	cmd.Flag.StringVar(&app.MdParamFuncName, "md_param_func", "Funcs_Main_POS_Both_Prop", "MD param func types: ["+types.AllParamFuncNames+"]")
	cmd.Flag.StringVar(&app.MdModelName, "md_model_name", "md_model_temp_i9.b64", "MD model file")
	cmd.Flag.StringVar(&app.DepModelName, "dep_model_name", "dep_zeager_model_temp_i18.b64", "Dep model file")
	cmd.Flag.StringVar(&app.DepFeaturesFile, "dep_features", "zhangnivre2011.yaml", "Dep features file")
	cmd.Flag.StringVar(&app.DepLabelsFile, "dep_labels", "hebtb.labels.conf", "Dep labels file")
	cmd.Flag.StringVar(&app.JointFeaturesFile, "joint_features", "jointzeager.yaml", "Joint features file")
	cmd.Flag.StringVar(&app.JointModelFile, "joint_model_name", "joint_arc_zeager_model_temp_i33.b64", "Joint model file")
	cmd.Flag.StringVar(&app.JointStrategy, "joint_strategy", "ArcGreedy", "Joint Strategy: ["+joint.JointStrategies+"]")
	cmd.Flag.StringVar(&app.OracleStrategy, "joint_oracle_strategy", "ArcGreedy", "Oracle Strategy: ["+joint.OracleStrategies+"]")
	return cmd
}

func StartAPIServer(cmd *commander.Command, args []string) error {
	HebrewMorphAnalyazerInitialize(cmd, args)
	MorphDisambiguatorInitialize(cmd, args)
	DepParserInitialize(cmd, args)
	JointParserInitialize()
	router = mux.NewRouter()
	router.HandleFunc("/yap/heb/ma", HebrewMorphAnalyzerHandler)
	router.HandleFunc("/yap/heb/md", MorphDisambiguatorHandler)
	router.HandleFunc("/yap/heb/dep", DepParserHandler)
	router.HandleFunc("/yap/heb/pipeline", HebrewPipelineHandler)
	router.HandleFunc("/yap/heb/joint", HebrewJointHandler)
	log.Fatal(http.ListenAndServe(":8000", router))
	return nil
}
