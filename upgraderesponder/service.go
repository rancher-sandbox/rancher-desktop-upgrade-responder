package upgraderesponder

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	influxcli "github.com/influxdata/influxdb/client/v2"
	maxminddb "github.com/oschwald/maxminddb-golang"
	"github.com/pkg/errors"

	rd "github.com/longhorn/upgrade-responder/rancherdesktop"
	"github.com/longhorn/upgrade-responder/utils"
)

const (
	AppMinimalVersion = "v0.0.1"

	InfluxDBMeasurement              = "upgrade_request"
	InfluxDBMeasurementDownSampling  = "upgrade_request_down_sampling"
	InfluxDBMeasurementByAppVersion  = "by_app_version_down_sampling"
	InfluxDBMeasurementByCountryCode = "by_country_code_down_sampling"

	InfluxDBContinuousQueryDownSampling  = "cq_upgrade_request_down_sampling"
	InfluxDBContinuousQueryByAppVersion  = "cq_by_app_version_down_sampling"
	InfluxDBContinuousQueryByCountryCode = "cq_by_country_code_down_sampling"
)

var (
	InfluxDBPrecisionNanosecond   = "ns" // ns is good for counting nodes
	InfluxDBDatabase              = "upgrade_responder"
	InfluxDBContinuousQueryPeriod = "1h"

	InfluxDBTagAppVersion             = "app_version"
	InfluxDBTagKubernetesVersion      = "kubernetes_version"
	InfluxDBTagLocationCity           = "city"
	InfluxDBTagLocationCountry        = "country"
	InfluxDBTagLocationCountryISOCode = "country_isocode"

	HTTPHeaderXForwardedFor = "X-Forwarded-For"
	ValueFieldKey           = "value" // A dummy InfluxDB field used to count the number of points
	ValueFieldValue         = 1
)

type Server struct {
	done chan struct{}
	// The set of versions that is returned when the client does
	// not include the information required to make an InstanceInfo.
	DefaultVersions []rd.Version
	// Maps Rules to a slice of versions with Version.Supported
	// precomputed according to Rule.Constraints.
	PrecomputedVersions []PrecomputedVersion
	influxClient        influxcli.Client
	db                  *maxminddb.Reader
	dbCache             *DBCache
}

type PrecomputedVersion struct {
	Rule     rd.Rule
	Versions []rd.Version
}

type Location struct {
	City    string `json:"city"`
	Country struct {
		Name    string
		ISOCode string
	} `json:"country"`
}

type CheckUpgradeResponse struct {
	Versions                 []rd.Version `json:"versions"`
	RequestIntervalInMinutes int          `json:"requestIntervalInMinutes"`
}

func NewServer(done chan struct{}, applicationName, configFile, influxURL, influxUser, influxPass, queryPeriod, geodb string, cacheSyncInterval, cacheSize int) (*Server, error) {
	InfluxDBDatabase = applicationName + "_" + InfluxDBDatabase
	InfluxDBContinuousQueryPeriod = queryPeriod

	// Parse and validate config
	path := filepath.Clean(configFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "fail to open configFile")
	}
	defer f.Close()
	var config rd.ResponseConfig
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s := &Server{
		done:            done,
		DefaultVersions: config.Versions,
	}
	if err := s.generatePrecomputedVersions(config); err != nil {
		return nil, fmt.Errorf("failed to generate precomputed versions: %w", err)
	}

	db, err := maxminddb.Open(geodb)
	if err != nil {
		return nil, errors.Wrap(err, "fail to open geodb file")
	}
	s.db = db
	logrus.Debugf("GeoDB opened")

	if influxURL != "" {
		cfg := influxcli.HTTPConfig{
			Addr:               influxURL,
			InsecureSkipVerify: true,
		}
		if influxUser != "" {
			cfg.Username = influxUser
		}
		if influxPass != "" {
			cfg.Password = influxPass
		}
		c, err := influxcli.NewHTTPClient(cfg)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("InfluxDB connection established")

		s.influxClient = c
		if err := s.initDB(); err != nil {
			return nil, err
		}
	}
	go func() {
		<-done
		if err := s.db.Close(); err != nil {
			logrus.Debugf("Failed to close geodb: %v", err)
		} else {
			logrus.Debugf("Geodb connection closed")
		}
		if s.influxClient != nil {
			if err := s.influxClient.Close(); err != nil {
				logrus.Debugf("Failed to close InfluxDB connection: %v", err)
			} else {
				logrus.Debug("InfluxDB connection closed")
			}
		}
	}()

	dbCache, err := NewDBCache(InfluxDBDatabase, InfluxDBPrecisionNanosecond, time.Duration(cacheSyncInterval)*time.Second, cacheSize, s.influxClient)
	if err != nil {
		return nil, err
	}
	s.dbCache = dbCache
	go s.dbCache.Run(done)

	return s, nil
}

func (s *Server) initDB() error {
	if err := s.createDB(InfluxDBDatabase); err != nil {
		return err
	}
	if err := s.createContinuousQueries(InfluxDBDatabase); err != nil {
		return err
	}
	return nil
}

func (s *Server) createDB(name string) error {
	q := influxcli.NewQuery("CREATE DATABASE "+name, "", "")
	response, err := s.influxClient.Query(q)
	if err != nil {
		return err
	}
	if response.Error() != nil {
		return response.Error()
	}
	logrus.Debugf("Database %v is either created or already exists", name)
	return nil
}

func (s *Server) createContinuousQueries(dbName string) error {
	queryStrings := map[string]string{}

	queryStrings[InfluxDBContinuousQueryDownSampling] = fmt.Sprintf("CREATE CONTINUOUS QUERY %v ON %v BEGIN SELECT count(%v) as total INTO %v FROM %v GROUP BY time(%v) END",
		InfluxDBContinuousQueryDownSampling, dbName, utils.ToSnakeCase(ValueFieldKey), InfluxDBMeasurementDownSampling, InfluxDBMeasurement, InfluxDBContinuousQueryPeriod)
	queryStrings[InfluxDBContinuousQueryByAppVersion] = fmt.Sprintf("CREATE CONTINUOUS QUERY %v ON %v BEGIN SELECT count(%v) as total INTO %v FROM %v GROUP BY time(%v),%v END",
		InfluxDBContinuousQueryByAppVersion, dbName, utils.ToSnakeCase(ValueFieldKey), InfluxDBMeasurementByAppVersion, InfluxDBMeasurement, InfluxDBContinuousQueryPeriod, InfluxDBTagAppVersion)
	queryStrings[InfluxDBContinuousQueryByCountryCode] = fmt.Sprintf("CREATE CONTINUOUS QUERY %v ON %v BEGIN SELECT count(%v) as total INTO %v FROM %v GROUP BY time(%v),%v END",
		InfluxDBContinuousQueryByCountryCode, dbName, utils.ToSnakeCase(ValueFieldKey), InfluxDBMeasurementByCountryCode, InfluxDBMeasurement, InfluxDBContinuousQueryPeriod, InfluxDBTagLocationCountryISOCode)

	for queryName, queryString := range queryStrings {
		query := influxcli.NewQuery(queryString, "", "")
		response, err := s.influxClient.Query(query)
		if err != nil {
			return err
		}
		if err := response.Error(); err != nil {
			if utils.IsAlreadyExistsError(err) {
				logrus.Debugf("The continuous query %v is already exists and cannot be modified. If you modified --query-period, please manually drop the continuous query %v from the database %v and retry", queryName, queryName, dbName)
			}
			return err
		}
		logrus.Debugf("Created continuous query %v", queryName)
	}
	return nil
}

func (s *Server) HealthCheck(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func (s *Server) CheckUpgrade(rw http.ResponseWriter, req *http.Request) {
	var (
		err       error
		checkReq  rd.CheckUpgradeRequest
		checkResp *CheckUpgradeResponse
	)

	defer func() {
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
	}()

	if err = json.NewDecoder(req.Body).Decode(&checkReq); err != nil {
		return
	}

	s.recordRequest(req, &checkReq)

	checkResp, err = s.GenerateCheckUpgradeResponse(&checkReq)
	if err != nil {
		logrus.Errorf("Failed to GenerateCheckUpgradeResponse: %v", err)
		return
	}

	if err = respondWithJSON(rw, checkResp); err != nil {
		logrus.Errorf("Failed to repsondWithJSON: %v", err)
		return
	}
	return
}

func respondWithJSON(rw http.ResponseWriter, obj interface{}) error {
	response, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrapf(err, "fail to marshal %v", obj)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(response)
	return err
}

func (s *Server) GenerateCheckUpgradeResponse(request *rd.CheckUpgradeRequest) (*CheckUpgradeResponse, error) {
	resp := &CheckUpgradeResponse{}

	instanceInfo, err := rd.NewInstanceInfo(request)
	if err != nil {
		logrus.Debugf("could not parse request %+v as InstanceInfo: %s", request, err)
		resp.Versions = s.DefaultVersions
	} else {
		logrus.Debugf("parsed request into InstanceInfo %+v: %s", request, err)
		for _, precomp := range s.PrecomputedVersions {
			if precomp.Rule.Test(instanceInfo) {
				resp.Versions = precomp.Versions
				break
			}
		}
		if len(resp.Versions) == 0 {
			resp.Versions = s.DefaultVersions
		}
	}

	d, err := time.ParseDuration(InfluxDBContinuousQueryPeriod)
	if err != nil {
		logrus.Errorf("fail to parse InfluxDBContinuousQueryPeriod while building upgrade response: %v", err)
		resp.RequestIntervalInMinutes = 60
	} else {
		resp.RequestIntervalInMinutes = int(d.Minutes())
	}

	return resp, nil
}

type locationRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		Names   map[string]string `maxminddb:"names"`
		ISOCode string            `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func (s *Server) getLocation(addr string) (*Location, error) {
	var (
		record locationRecord
		loc    Location
	)
	ip := net.ParseIP(addr)

	err := s.db.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}

	loc.City = record.City.Names["en"]
	loc.Country.Name = record.Country.Names["en"]
	loc.Country.ISOCode = record.Country.ISOCode
	return &loc, nil
}

//func canonializeField(name string) string {
//	return strings.Replace(strings.ToLower(HTTPHeaderRequestID), "-", "_", -1)
//}

// Don't need to return error to the requester
func (s *Server) recordRequest(httpReq *http.Request, req *rd.CheckUpgradeRequest) {
	xForwaredFor := httpReq.Header[HTTPHeaderXForwardedFor]
	publicIP := ""
	l := len(xForwaredFor)
	if l > 0 {
		// rightmost IP must be the public IP
		publicIP = xForwaredFor[l-1]
	}

	// We use IP to find the location but we don't store IP
	loc, err := s.getLocation(publicIP)
	if err != nil {
		logrus.Error("Failed to get location for one ip")
	}

	if s.influxClient != nil {
		var (
			err error
			pt  *influxcli.Point
		)
		defer func() {
			if err != nil {
				logrus.Errorf("Failed to recordRequest: %v", err)
			}
		}()

		tags := map[string]string{
			InfluxDBTagAppVersion: req.AppVersion,
		}
		for k, v := range req.ExtraInfo {
			tags[utils.ToSnakeCase(k)] = v
		}
		fields := map[string]interface{}{
			utils.ToSnakeCase(ValueFieldKey): ValueFieldValue,
		}
		if loc != nil {
			tags[InfluxDBTagLocationCity] = loc.City
			tags[InfluxDBTagLocationCountry] = loc.Country.Name
			tags[InfluxDBTagLocationCountryISOCode] = loc.Country.ISOCode
		}
		pt, err = influxcli.NewPoint(InfluxDBMeasurement, tags, fields, time.Now())
		if err != nil {
			return
		}

		s.dbCache.AddPoint(pt)
	}
}

func (s *Server) generatePrecomputedVersions(config rd.ResponseConfig) error {
	rulesWithPrecomputedVersions := make([]PrecomputedVersion, 0, len(config.Rules))
	for _, rule := range config.Rules {
		precomputedVersions := make([]rd.Version, 0, len(config.Versions))
		for _, version := range config.Versions {
			precomputedVersion := version
			supported, err := rule.Supported(version)
			if err != nil {
				return fmt.Errorf("failed to compute Supported for Rule %+v and Version %q: %w", rule, version.Name, err)
			}
			precomputedVersion.Supported = supported
			precomputedVersions = append(precomputedVersions, precomputedVersion)
		}
		newElement := PrecomputedVersion{
			Rule:     rule,
			Versions: precomputedVersions,
		}
		rulesWithPrecomputedVersions = append(rulesWithPrecomputedVersions, newElement)
	}

	s.PrecomputedVersions = rulesWithPrecomputedVersions
	return nil
}
